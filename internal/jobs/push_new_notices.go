package jobs

import (
	"time"

	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

// PushNewNotices checks and pushes new notices for all users
// TODO: use goroutine to send messages concurrently? (no, it's not worth it, FIB API server has poor concurrency, better reuse connection)
func PushNewNotices() {
	logger := log.WithField("job", "PushNewNotices")

	userIDs, err := db.GetAllUserIDs()
	if err != nil {
		logger.Errorf("failed to get all user IDs: %v", err)
		return
	}

	waitUntilSecond5() // FIXME: hacky

	var checkedUserCount, totalFetchedCount, totalSentCount uint32
	start := time.Now()
	for _, userID := range userIDs {
		userLogger := logger.WithField("UID", userID)
		client := bot.NewClient(userID)
		if client == nil {
			// possible database corruption
			userLogger.Error("failed to create client")
			// try sending a message to the user to ask them to re-login to fix it
			_ = bot.SendMessage(userID, &bot.ErrorMessage{
				Text: locales.Get("default").FIBAPIAuthorizationExpiredMessage,
			})
			if err = db.DelUser(userID); err != nil {
				logger.Errorf("failed to delete user %d: %v", userID, err)
			}
			continue
		}

		var newNotices []bot.NoticeMessage
		newNotices, err = client.GetNewNotices()
		if err != nil {
			userLogger.Errorf("failed to get new notices: %v", err)
			if err == fibapi.ErrAuthorizationExpired {
				// notify the user that their FIB API authorization has expired
				if bot.SendMessage(userID, &bot.ErrorMessage{
					Text: locales.Get(client.User.LanguageCode).FIBAPIAuthorizationExpiredMessage,
				}) != nil {
					// delete them from DB if the notification was sent successfully
					if err = db.DelUser(userID); err != nil {
						logger.Errorf("failed to delete user %d: %v", userID, err)
					}
				}
			}
			continue
		}
		checkedUserCount++

		if len(newNotices) == 0 { // nothing new
			continue
		}
		totalFetchedCount += uint32(len(newNotices))
		var userSentCount uint32
		for _, n := range newNotices {
			if bot.SendMessage(userID, &n) != nil {
				userSentCount++
				totalSentCount++
			}
		}
		userLogger.Infof("sent %d/%d new notices", userSentCount, len(newNotices))
	}

	logger.Infof("checked %d/%d users and sent %d/%d new notices in %s",
		checkedUserCount, len(userIDs),
		totalSentCount, totalFetchedCount,
		time.Since(start))
}

// waitUntilSecond5 waits until the current time is at least 5 seconds into the minute
// its purpose is to avoid fetching notices too early (missing new notices) in case of the clock on FIB API server is slower
func waitUntilSecond5() {
	now := time.Now()
	if now.Second() < 5 {
		time.Sleep(time.Duration(5-now.Second()) * time.Second)
	}
}

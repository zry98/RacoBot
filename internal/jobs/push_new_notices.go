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
// TODO: use goroutine to send messages concurrently?
func PushNewNotices() {
	logger := log.WithField("job", "PushNewNotices")
	var checkedUserCount, sentMessageCount uint32

	userIDs, err := db.GetAllUserIDs()
	if err != nil {
		logger.Errorf("failed to get all user IDs: %v", err)
		return
	}

	waitUntilSecond5() // FIXME: hacky

	start := time.Now()
	for _, userID := range userIDs {
		userLogger := logger.WithField("UID", userID)
		client := bot.NewClient(userID)
		if client == nil {
			// possible database corruption
			userLogger.Error("failed to create client")
			// ask the user to re-login to fix it
			bot.SendMessage(userID, &bot.ErrorMessage{locales.Get("default").FIBAPIAuthorizationExpiredMessage})
			if err = db.DeleteUser(userID); err != nil {
				logger.Errorf("failed to delete user %d: %v", userID, err)
			}
			continue
		}

		newNotices, e := client.GetNewNotices()
		if e != nil {
			if e == fibapi.ErrAuthorizationExpired {
				// notify the user that their FIB API authorization has expired and delete them from DB
				userLogger.Info("token has expired")
				bot.SendMessage(userID, &bot.ErrorMessage{locales.Get(client.User.LanguageCode).FIBAPIAuthorizationExpiredMessage})
				if err = db.DeleteUser(userID); err != nil {
					logger.Errorf("failed to delete user %d: %v", userID, err)
				}
			} else {
				userLogger.Errorf("failed to get new notices: %v", e)
			}
			continue
		}
		userLogger.Infof("fetched %d new notices", len(newNotices))

		for _, n := range newNotices {
			bot.SendMessage(userID, &n)
			userLogger.Infof("sent new notice %d", n.ID)
			sentMessageCount++
		}

		checkedUserCount++
	}

	logger.Infof("checked %d/%d users and sent %d messages in %s",
		checkedUserCount, len(userIDs), sentMessageCount, time.Since(start))
}

// waitUntilSecond5 waits until the current time is at least 5 seconds into the minute
// its purpose is to avoid fetching notices too early (missing new notices) in case of the clock on FIB API server is slower
func waitUntilSecond5() {
	now := time.Now()
	if now.Second() < 5 {
		time.Sleep(time.Duration(5-now.Second()) * time.Second)
	}
}

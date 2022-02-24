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
func PushNewNotices() { // TODO: use goroutine to send messages concurrently?
	logger := log.WithField("Job", "PushNewNotices")
	logger.Info("Started")

	var checkedUserCount, sentMessageCount uint

	userIDs, err := db.GetUserIDs()
	if err != nil {
		logger.Error(err)
		return
	}

	waitUntilSecond5() // FIXME: hacky

	for _, userID := range userIDs {
		userLogger := logger.WithField("UID", userID)
		client := bot.NewClient(userID)
		if client == nil {
			// possible database corruption
			userLogger.Error("Failed to create client")
			// ask the user to re-login to fix it
			bot.SendMessage(userID, locales.Get("default").FIBAPIAuthorizationExpiredErrorMessage)
			if err = db.DeleteUser(userID); err != nil {
				logger.Error(err)
			}
			continue
		}

		var newNotices []bot.NoticeMessage
		newNotices, err = client.GetNewNotices()
		if err != nil {
			if err == fibapi.ErrAuthorizationExpired {
				// notify the user that their FIB API authorization is expired and delete them from DB
				userLogger.Info("FIB API token expired")
				bot.SendMessage(userID, locales.Get(client.User.LanguageCode).FIBAPIAuthorizationExpiredErrorMessage)
				if err = db.DeleteUser(userID); err != nil {
					logger.Error(err)
				}
			} else {
				userLogger.Errorf("Error fetching new notices: %s", err.Error())
			}
			continue
		}
		userLogger.Infof("Fetched %d new notices", len(newNotices))

		for _, n := range newNotices {
			bot.SendMessage(userID, &n)
			userLogger.Infof("Sent new notice %d", n.ID)
			sentMessageCount++
		}

		checkedUserCount++
	}

	logger.Infof("Done! total checked users: %d, total sent messages: %d", checkedUserCount, sentMessageCount)
}

// waitUntilSecond5 waits until the current time is at least 5 seconds into the minute
// its purpose is to avoid fetching notices too early (missing new notices) in case of the clock on FIB API server is slower
func waitUntilSecond5() {
	now := time.Now()
	if now.Second() < 5 {
		time.Sleep(time.Duration(5-now.Second()) * time.Second)
	}
}

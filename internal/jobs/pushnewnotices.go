package jobs

import (
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

	var checkedUserCount, sentMessageCount int

	userIDs, err := db.GetUserIDs()
	if err != nil {
		logger.Error(err)
		return
	}

	for _, userID := range userIDs {
		checkedUserCount++
		client := bot.NewClient(userID)
		if client == nil {
			bot.SendMessage(userID, locales.Get("en").FIBAPIAuthorizationExpiredErrorMessage)
			if err = db.DeleteUser(userID); err != nil {
				logger.Error(err)
			}
			continue
		}

		newNotices, err := client.GetNewNotices()
		if err != nil {
			if err == fibapi.ErrAuthorizationExpired {
				logger.Infof("FIB API token expired for user %d", userID)
				bot.SendMessage(userID, locales.Get(client.User.LanguageCode).FIBAPIAuthorizationExpiredErrorMessage)
				if err = db.DeleteUser(userID); err != nil {
					logger.Error(err)
				}
			} else {
				logger.Errorf("Error getting new notices for user %d, detail: %s", userID, err.Error())
			}
			continue
		}
		logger.Infof("Fetched %d new notices for user %d", len(newNotices), userID)

		for _, n := range newNotices {
			bot.SendMessage(userID, &n)
			logger.Infof("Sent new notice %d to user %d", n.ID, userID)
			sentMessageCount++
		}
	}
	logger.Infof("Done, total checked users: %d, total sent messages: %d", checkedUserCount, sentMessageCount)
}

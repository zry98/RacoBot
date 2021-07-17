package jobs

import (
	"RacoBot/internal/locales"
	"time"

	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

// JobsConfig represents a configuration for the jobs
type JobsConfig struct {
	PushNewNoticesCron string `toml:"push_new_notices_cron"`
}

// Init initializes the jobs
func Init(config JobsConfig) {
	if config.PushNewNoticesCron != "" {
		go RunJobs(config)
	}
}

// RunJobs initializes the scheduler and starts the jobs
func RunJobs(config JobsConfig) {
	// TODO: use MQ?
	defer func() {
		if err := recover(); err != nil {
			log.Warn("error recovered in RunJobs ", err)
			if err == bot.UserNotFoundError {

			}
		}
	}()

	tzMadrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		log.Error(err)
		return
	}

	s := gocron.NewScheduler(tzMadrid)
	s.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
	_, err = s.Cron(config.PushNewNoticesCron).Tag("PushNewNotices").Do(PushNewNotices)
	if err != nil {
		log.Error(err)
		return
	}

	s.StartAsync()
}

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
			if err == fibapi.AuthorizationExpiredError {
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
			bot.SendMessage(userID, n)
			logger.Infof("Sent new notice %d to user %d", n.ID, userID)
			sentMessageCount++
		}
	}
	logger.Infof("Done, total checked users: %d, total sent messages: %d", checkedUserCount, sentMessageCount)
}

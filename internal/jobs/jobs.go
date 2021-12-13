package jobs

import (
	"time"

	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
)

// Config represents a configuration for the jobs
type Config struct {
	PushNewNoticesCronExp string `toml:"push_new_notices_cron"`
}

// Init initializes the jobs
func Init(config Config) {
	if config.PushNewNoticesCronExp != "" {
		go RunJobs(config)
	}
}

// RunJobs initializes the scheduler and starts the jobs
func RunJobs(config Config) {
	// TODO: use MQ?
	defer func() {
		if err := recover(); err != nil {
			log.Warn("error recovered in RunJobs: ", err)
		}
	}()

	tzMadrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		log.Error(err)
		return
	}

	s := gocron.NewScheduler(tzMadrid)
	s.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
	_, err = s.Cron(config.PushNewNoticesCronExp).Tag("PushNewNotices").Do(PushNewNotices)
	if err != nil {
		log.Error(err)
		return
	}

	s.StartAsync()
}

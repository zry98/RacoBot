package jobs

import (
	"time"

	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
)

// Config represents a configuration for the jobs
type Config struct {
	PushNewNoticesCronExp    string `toml:"push_new_notices_cron"`
	CacheSubjectCodesCronExp string `toml:"cache_subject_codes_cron"`
}

var scheduler *gocron.Scheduler

// Init initializes the jobs scheduler
func Init(config Config) {
	tzMadrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		panic(err)
	}

	scheduler = gocron.NewScheduler(tzMadrid)
	scheduler.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
	addJobs(config)
	scheduler.StartAsync()
}

// addJobs adds the jobs to the scheduler
func addJobs(config Config) {
	if config.PushNewNoticesCronExp != "" {
		_, err := scheduler.Cron(config.PushNewNoticesCronExp).Tag("PushNewNotices").Do(PushNewNotices)
		if err != nil {
			log.Errorf("failed to schedule PushNewNotices: %v", err)
		}
	}
	if config.CacheSubjectCodesCronExp != "" {
		_, err := scheduler.Cron(config.CacheSubjectCodesCronExp).Tag("CacheSubjectCodes").Do(CacheSubjectCodes)
		if err != nil {
			log.Errorf("failed to schedule CacheSubjectCodes: %v", err)
		}
	}
}

package job

import (
	"time"

	"github.com/go-co-op/gocron/v2"
	log "github.com/sirupsen/logrus"
)

// Config represents a configuration for the jobs
type Config struct {
	PushNewNoticesCronExp    string `toml:"push_new_notices_cron"`
	CacheSubjectCodesCronExp string `toml:"cache_subject_codes_cron"`
}

var scheduler gocron.Scheduler

// Init initializes the jobs scheduler
func Init(config Config) {
	tzMadrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		panic(err)
	}

	scheduler, err = gocron.NewScheduler(
		gocron.WithLocation(tzMadrid),
		gocron.WithLimitConcurrentJobs(1, gocron.LimitModeReschedule),
	)
	if err != nil {
		log.Fatalf("failed to create jobs scheduler: %v", err)
	}
	addJobs(config)
	scheduler.Start()
}

// Stop stops the jobs scheduler
func Stop() {
	if scheduler != nil {
		if err := scheduler.Shutdown(); err != nil {
			log.Errorf("failed to shutdown jobs scheduler: %v", err)
		}
	}
	log.Debug("jobs scheduler stopped")
}

// addJobs adds the jobs to the scheduler
func addJobs(config Config) {
	if config.PushNewNoticesCronExp != "" {
		if _, err := scheduler.NewJob(
			gocron.CronJob(config.PushNewNoticesCronExp, false),
			gocron.NewTask(PushNewNotices),
			gocron.WithName("PushNewNotices"),
		); err != nil {
			log.Errorf("failed to schedule PushNewNotices: %v", err)
		}
	}
	if config.CacheSubjectCodesCronExp != "" {
		if _, err := scheduler.NewJob(
			gocron.CronJob(config.CacheSubjectCodesCronExp, false),
			gocron.NewTask(CacheSubjectCodes),
			gocron.WithName("CacheSubjectCodes"),
		); err != nil {
			log.Errorf("failed to schedule CacheSubjectCodes: %v", err)
		}
	}
}

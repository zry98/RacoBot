package jobs

import (
	log "github.com/sirupsen/logrus"
	"time"

	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

// CacheSubjectCodes pulls all subject UPC codes from FIB Public API and stores them in the database
func CacheSubjectCodes() {
	logger := log.WithField("job", "CacheSubjectCodes")

	start := time.Now()
	subjects, err := fibapi.GetPublicSubjects()
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Infof("fetched %d subjects", len(subjects))

	codes := make(map[string]uint32, len(subjects))
	for _, s := range subjects {
		codes[s.ID] = s.UPCCode
	}
	if err = db.PutSubjectUPCCodes(codes); err != nil {
		logger.Errorf("failed to put subject codes: %v", err)
		return
	}

	logger.Infof("cached %d subject codes in %s", len(codes), time.Since(start))
}

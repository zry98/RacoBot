package jobs

import (
	"time"

	log "github.com/sirupsen/logrus"

	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

// CacheSubjectCodes pulls all subject UPC codes from public FIB API and stores them in the database
func CacheSubjectCodes() {
	logger := log.WithField("job", "CacheSubjectCodes")

	start := time.Now()
	subjects, err := fibapi.GetPublicSubjects()
	if err != nil {
		logger.Errorf("failed to get subjects: %v", err)
		return
	}
	logger.Infof("fetched %d subjects in %v", len(subjects), time.Since(start))

	if err = db.DeleteAllSubjectUPCCodes(); err != nil {
		logger.Errorf("failed to purge subject codes: %v", err)
		return
	}

	start = time.Now()
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

package jobs

import (
	log "github.com/sirupsen/logrus"

	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

// PullSubjectCodes pulls all subject UPC codes from FIB Public API and stores them in the database
func PullSubjectCodes() {
	logger := log.WithField("Job", "PullSubjectCodes")
	logger.Info("Started")

	subjects, err := fibapi.GetPublicSubjects()
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Infof("Fetched %d subjects", len(subjects))

	codes := make(map[string]uint32, len(subjects))
	for _, s := range subjects {
		codes[s.ID] = s.UPCCode
	}
	if err = db.PutSubjectUPCCodes(codes); err != nil {
		logger.Error(err)
		return
	}

	logger.Info("Done!")
}

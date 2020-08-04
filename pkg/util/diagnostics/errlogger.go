package diagnostics

import log "github.com/sirupsen/logrus"

func LogPublishErrors(msg string, errorList []error) {
	for _, err := range errorList {
		log.WithError(err).Debug(msg)
	}
}

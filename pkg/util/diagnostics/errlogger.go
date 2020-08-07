package diagnostics

import log "github.com/sirupsen/logrus"

// LogPublishErrors will log errors for a message. This method is mostly used for debug messages from rpcBus.Publish return errList
func LogPublishErrors(msg string, errorList []error) {
	for _, err := range errorList {
		log.WithError(err).Debug(msg)
	}
}

package logging

import (
	"os"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	log "github.com/sirupsen/logrus"
)

// InitLog initializes a logrus global instance
func InitLog(logFile *os.File) {
	// apply logger level from configurations
	SetToLevel(cfg.Get().Logger.Level)
	log.SetOutput(logFile)
}

// SetToLevel sets logrus instance to a specified level
func SetToLevel(l string) {
	level, err := log.ParseLevel(l)
	if err == nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(log.TraceLevel)
		log.Warnf("Parse logger level from config err: %v", err)
	}
}

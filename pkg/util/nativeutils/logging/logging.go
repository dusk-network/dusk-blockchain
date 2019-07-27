package logging

import (
	"os"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	log "github.com/sirupsen/logrus"
)

func InitLog(logFile *os.File) {
	// apply logger level from configurations
	SetToLevel(cfg.Get().Logger.Level)
	log.SetOutput(logFile)
}

func SetToLevel(l string) {
	level, err := log.ParseLevel(l)
	if err == nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(log.TraceLevel)
		log.Warnf("Parse logger level from config err: %v", err)
	}
}

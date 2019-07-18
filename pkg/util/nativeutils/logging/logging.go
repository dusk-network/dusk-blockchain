package logging

import (
	"os"

	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
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

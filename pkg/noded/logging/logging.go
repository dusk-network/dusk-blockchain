package logging

import (
	log "github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/noded/config"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	userHomeDuskDir = "/.dusk"
)

// ConfigureLogging configurates the logging
func ConfigureLogging() error {

	// Get the logging config.
	logFilepath := config.EnvNetCfg.Log.FilePath
	level := config.EnvNetCfg.Log.Level

	// Create a log file if not yet exist
	logFilepath = createLogFile(logFilepath, "net")

	dir := filepath.Dir(logFilepath)
	base := filepath.Base(logFilepath)
	os.MkdirAll(dir, os.ModePerm)

	logfile := filepath.Join(dir, base)
	logger := createRollingFileLogger(logfile)

	mw := io.MultiWriter(os.Stdout, &logger)
	log.SetOutput(mw)

	logLevel, err := log.ParseLevel(strings.ToUpper(level))
	if err != nil {
		return err
	}
	log.SetLevel(logLevel)

	// Use a custom formatter TODO: ext. configuration
	log.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat: config.EnvNetCfg.Log.TimestampFormat,
		FullTimestamp:   true,
		ForceFormatting: true,
		ForceColors:     config.EnvNetCfg.Log.Tty,
	})

	return nil
}

func createLogFile(logFilepath, envNet string) string {
	var err error

	if logFilepath == "" {
		logFilepath = UserHomeDir() + userHomeDuskDir + "/" + strings.ToLower(envNet) + "/noded/noded.log"
	}

	// Stat returns file info. It will return an error if there is no log file.
	_, err = os.Stat(logFilepath)

	// Create log file if not exists
	var file *os.File
	if os.IsNotExist(err) {
		file, _ = os.Create(logFilepath)
		defer file.Close()
	}

	return logFilepath
}

func createRollingFileLogger(logfile string) lumberjack.Logger {
	rollingFileLogger := lumberjack.Logger{
		Filename:   logfile,
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	}

	return rollingFileLogger
}

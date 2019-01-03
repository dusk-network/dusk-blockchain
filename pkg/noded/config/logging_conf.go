package config

import (
	"github.com/sirupsen/logrus"
	cnf "github.com/spf13/viper"
	"github.com/x-cray/logrus-prefixed-formatter"
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
func configureLogging() error {

	// Get the logging config.
	logFilepath := cnf.GetString("net.logging.filepath")
	level := cnf.GetString("net.logging.level")

	// Create a log file if not yet exist
	logFilepath = createLogFile(logFilepath, "net")

	dir := filepath.Dir(logFilepath)
	base := filepath.Base(logFilepath)
	os.MkdirAll(dir, os.ModePerm)

	logfile := filepath.Join(dir, base)
	logger := createRollingFileLogger(logfile)

	mw := io.MultiWriter(os.Stdout, &logger)
	logrus.SetOutput(mw)

	logLevel, err := logrus.ParseLevel(strings.ToUpper(level))
	if err != nil {
		return err
	}
	logrus.SetLevel(logLevel)

	// Use a custom formatter TODO: ext. configuration
	logrus.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat: cnf.GetString("net.logging.timestampformat"),
		FullTimestamp:   true,
		ForceFormatting: true,
		ForceColors:     cnf.GetBool("net.logging.tty"),
	})

	return nil
}

func createLogFile(logFilepath, envNet string) string {
	var err error

	if logFilepath == "" {
		logFilepath = userHomeDir() + userHomeDuskDir + "/" + strings.ToLower(envNet) + "/noded/noded.log"
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

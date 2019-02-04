package config

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"os"
)

// EnvNetCfg is a var to be able to read configuration globally
var EnvNetCfg EnvNetConfig

// EnvNetConfig holds the configuration for an network environment like TestNet, MainNet etc.
// Values can be set by the user in cmd/config/config.json.
type EnvNetConfig struct {
	Magic    uint32
	Log      Log
	Peer     Peer
	Monitor  Monitor
	Database Database
}

// Log holds all configuration related to logging
type Log struct {
	FilePath string
	// Log level (log levels: trace, debug, info, warning, error, fatal and panic)
	Level string
	// See: https://golang.org/src/time/format.go
	TimestampFormat string
	// Write log file for a TTY device (to see coloured log text)
	Tty bool
}

// Peer holds all configuration related to the local peer
type Peer struct {
	Port        uint16
	DialTimeout uint16
	Seeds       []string
	// Minimum peers before logging error messaages (or any other future notification system)
	Min uint16
	// Maximum peers before disabling peers
	Max uint16
	// Minimum peers before sending 'getaddr' messages to find more peers
	MinGetAddr uint16
}

// Monitor holds all configuration related to monitoring peer synchronisation thresholds
type Monitor struct {
	// Disable events for a certain time to be triggered again.
	EvtDisableDuration string
}

// Database holds all configuration related to the blockchain database
type Database struct {
	DirPath string
}

// LoadConfig reads and loads the configuration from cmd/config/config.json
func LoadConfig(file string) {
	var config EnvNetConfig
	configFile, err := os.Open(file)
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		log.Fatalf("Could not read configuration: %s", err.Error())
	}

	EnvNetCfg = config
}

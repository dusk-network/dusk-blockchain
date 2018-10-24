package main

import (
	"os"
	"path/filepath"
)

type Config struct {
	File string
	RPCUser string
	RPCPassword string
	RPCPort string
}

var appPath, _ = os.Executable()
var appDir = filepath.Dir(appPath)

func LoadConfig() (*Config, error) {
	// Initialize config object
	cfg := Config{
		File: filepath.Join(appDir, "dusk.conf"),
		RPCPort: "9999",
	}

	// TODO: Read from File field and parse into cfg

	return &cfg, nil
}

// TODO: Allow LoadConfig() from user-specified path

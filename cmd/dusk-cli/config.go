package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all the configurations for the CLI
type Config struct {
	File        string
	RPCUser     string
	RPCPassword string
	RPCPort     string
}

// LoadConfig will load the specified config file, parse command line flags
// and populate a Config struct, then return it.
func LoadConfig() (*Config, error) {
	// Retrieve binary path
	appPath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not retrieve binary path")
	}

	appDir := filepath.Dir(appPath)

	// Initialize config object
	cfg := Config{
		File: filepath.Join(appDir, "dusk.conf"),
	}

	// Parse flags and set filepath if specified
	flag.Parse()
	if *conf != "dusk.conf" {
		cfg.File = *conf
	}

	// Create config file if it doesn't exist yet
	if _, err := os.Stat(cfg.File); os.IsNotExist(err) {
		if err := CreateNewConfig(cfg.File); err != nil {
			return nil, err
		}
	}

	// Read file
	bs, err := ioutil.ReadFile(cfg.File)
	if err != nil {
		return nil, err
	}

	// Seperate the fields and populate the Config struct
	fields := strings.Split(string(bs), "\n")
	for _, field := range fields {
		switch {
		case strings.Contains(field, "rpcport="):
			cfg.RPCPort = strings.Trim(field, "rpcport=")
		case strings.Contains(field, "rpcuser="):
			cfg.RPCUser = strings.Trim(field, "rpcuser=")
		case strings.Contains(field, "rpcpass="):
			cfg.RPCPassword = strings.Trim(field, "rpcpass=")
		default:
			break
		}
	}

	// Parse flags if specified
	if *rpcuser != "dusk123" {
		cfg.RPCUser = *rpcuser
	}
	if *rpcpass != "duskpass" {
		cfg.RPCPassword = *rpcpass
	}
	if *rpcport != "9999" {
		cfg.RPCPort = *rpcport
	}

	return &cfg, nil
}

// CreateNewConfig will create a new config file on the disk with standard parameters.
// This function may be called if no config file is present on launch.
func CreateNewConfig(dest string) error {
	file, err := os.Create(dest)
	defer file.Close()
	if err != nil {
		return err
	}

	// Placeholder values
	fields := []string{"rpcuser=dusk123\n", "rpcpass=duskpass\n", "rpcport=9999"}
	for _, field := range fields {
		if _, err := file.Write([]byte(field)); err != nil {
			return err
		}
	}

	return nil
}

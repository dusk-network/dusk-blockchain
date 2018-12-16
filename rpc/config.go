package rpc

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all the options for the RPC server
type Config struct {
	File    string // Config filepath
	RPCUser string // Username
	RPCPass string // Password
	RPCPort string // Port to bind the RPC server on
}

// Load will load a config file into cfg.
func (cfg *Config) Load() error {
	// Retrieve binary path
	appPath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not retrieve binary path")
	}

	appDir := filepath.Dir(appPath)

	// Set filepath if not specified
	if cfg.File == "" {
		cfg.File = filepath.Join(appDir, "dusk.conf")
	}

	// Create config file if it doesn't exist yet in specified path
	if _, err := os.Stat(cfg.File); os.IsNotExist(err) {
		if err := CreateNewConfig(cfg.File); err != nil {
			return err
		}
	}

	// Read file
	bs, err := ioutil.ReadFile(cfg.File)
	if err != nil {
		return err
	}

	// Seperate the fields and populate the Config struct
	fields := strings.Split(string(bs), "\n")
	for _, field := range fields {
		switch {
		case strings.Contains(field, "rpcport="):
			cfg.RPCPort = strings.TrimPrefix(field, "rpcport=")
		case strings.Contains(field, "rpcuser="):
			cfg.RPCUser = strings.TrimPrefix(field, "rpcuser=")
		case strings.Contains(field, "rpcpass="):
			cfg.RPCPass = strings.TrimPrefix(field, "rpcpass=")
		default:
			break
		}
	}

	return nil
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

package config

import (
	cfg "github.com/spf13/viper"
	"log"
	"strings"
)

const (
	errConfigRead = "Error when reading config:\n%s"
)

// LoadConfig loads all the noded configuration from the noded.toml file.
// It's also aware of some default configuration settings.
func LoadConfig() error {

	cfg.SetConfigName("noded")            // Name of config file without file extension
	cfg.AddConfigPath("cmd/noded/config") // Path of config file
	cfg.AutomaticEnv()

	err := cfg.ReadInConfig()
	if err != nil {
		log.Fatalf(errConfigRead, err)
	}
	cfg.BindEnv("Env.Net", "DUSK.ENV.NET")

	envNet := cfg.GetString("Env.Net")
	if envNet == "" {
		log.Fatalf(errConfigRead,
			"Please set at least:\n"+
				" - config var 'env.net' or\n"+
				" - env var 'DUSK.ENV.NET' or\n"+
				" - cmdline arg 'env.net'.",
		)
	}

	cfg.RegisterAlias(envNet, "net")

	setDefaults(envNet)

	err = configureLogging()

	// TODO: Any other initialisation of external config

	return err
}

func setDefaults(envNet string) {
	// Logging
	cfg.SetDefault("net.logging.filepath", UserHomeDir()+userHomeDuskDir+"/"+strings.ToLower(envNet)+"/noded/noded.log")
	cfg.SetDefault("net.logging.level", "info")
	cfg.SetDefault("net.logging.timestampformat", "2006-01-02 15:04:05")
	cfg.SetDefault("net.logging.tty", false)
	// Peer
	cfg.SetDefault("net.peer.port", "10333") //TODO: To be decided
	cfg.SetDefault("net.peer.dialtimeout", 0)

	// Database
	cfg.SetDefault("net.database.dirpath", UserHomeDir()+userHomeDuskDir+"/"+strings.ToLower(envNet)+"/db")

	// Monitoring Events
	cfg.SetDefault("net.monitoring.evtDisableDuration", "1m")

}

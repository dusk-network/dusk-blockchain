package config

import (
	cnf "github.com/spf13/viper"
	"log"
	"strings"
)

const (
	errConfigRead = "Error when reading config:\n%s"
)

// LoadConfig loads all the noded configuration from the noded.toml file.
// It's also aware of some default configuration settings.
func LoadConfig() error {

	cnf.SetConfigName("noded")            // Name of config file without file extension
	cnf.AddConfigPath("cmd/noded/config") // Path of config file
	cnf.AutomaticEnv()

	err := cnf.ReadInConfig()
	if err != nil {
		log.Fatalf(errConfigRead, err)
	}
	cnf.BindEnv("Env.Net", "DUSK.ENV.NET")

	envNet := cnf.GetString("Env.Net")
	if envNet == "" {
		log.Fatalf(errConfigRead,
			"Please set at least:\n"+
				" - config var 'env.net' or\n"+
				" - env var 'DUSK.ENV.NET' or\n"+
				" - cmdline arg 'env.net'.",
		)
	}

	cnf.RegisterAlias(envNet, "net")

	setDefaults(envNet)

	err = configureLogging()

	// TODO: Any other initialisation of external config

	return err
}

func setDefaults(envNet string) {
	// Logging
	cnf.SetDefault("net.logging.filepath", userHomeDir()+userHomeDuskDir+"/"+strings.ToLower(envNet)+"/noded/noded.log")
	cnf.SetDefault("net.logging.level", "info")
	cnf.SetDefault("net.logging.timestampformat", "2006-01-02 15:04:05")
	cnf.SetDefault("net.logging.tty", false)
	// Peer
	cnf.SetDefault("net.peer.port", "10333") //TODO: To be decided
	cnf.SetDefault("net.peer.dialtimeout", 0)

	// Database
	cnf.SetDefault("net.database.dirpath", userHomeDir()+userHomeDuskDir+"/"+strings.ToLower(envNet)+"/db")

}

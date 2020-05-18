package ruskmock

import (
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/grpc"
)

// Config contains a list of settings that determine the behaviour
// of the `Server`.
type Config struct{}

// DefaultConfig returns the default configuration for the Rusk mock server.
func DefaultConfig() *Config {
	return &Config{}
}

// Server is a stand-in Rusk server, which can be used during any kind of
// testing. Its behaviour can be modified depending on the settings of the
// `Config` struct, contained in the `Server`, to simulate different types
// of scenarios on demand.
type Server struct {
	cfg *Config
}

// New returns a new Rusk mock server with the given config. If no config is
// passed, a default one is put into place.
func New(cfg *Config) (*Server, error) {
	grpcServer := grpc.NewServer()

	if cfg == nil {
		cfg = DefaultConfig()
	}

	srv := &Server{
		cfg: cfg,
	}

	rusk.RegisterRuskServer(grpcServer, srv)
	return srv, nil
}

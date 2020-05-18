package ruskmock

import (
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/grpc"
)

type Config struct{}

func DefaultConfig() *Config {
	return &Config{}
}

type Server struct {
	cfg *Config
}

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

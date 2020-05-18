package ruskmock

import (
	"context"

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
	if cfg == nil {
		cfg = DefaultConfig()
	}

	srv := &Server{
		cfg: cfg,
	}

	grpcServer := grpc.NewServer()
	rusk.RegisterRuskServer(grpcServer, srv)
	return srv, nil
}

func (s *Server) Echo(ctx context.Context, req *rusk.EchoRequest) (*rusk.EchoResponse, error) {
	return nil, nil
}

func (s *Server) ValidateStateTransition(ctx context.Context, req *rusk.ValidateStateTransitionRequest) (*rusk.ValidateStateTransitionResponse, error) {
	return nil, nil
}

func (s *Server) ExecuteStateTransition(ctx context.Context, req *rusk.ExecuteStateTransitionRequest) (*rusk.ExecuteStateTransitionResponse, error) {
	return nil, nil
}

func (s *Server) GenerateScore(ctx context.Context, req *rusk.GenerateScoreRequest) (*rusk.GenerateScoreResponse, error) {
	return nil, nil
}

func (s *Server) VerifyScore(ctx context.Context, req *rusk.VerifyScoreRequest) (*rusk.VerifyScoreResponse, error) {
	return nil, nil
}

func (s *Server) GenerateSecretKey(ctx context.Context, req *rusk.GenerateSecretKeyRequest) (*rusk.GenerateSecretKeyResponse, error) {
	return nil, nil
}

func (s *Server) Keys(ctx context.Context, req *rusk.SecretKey) (*rusk.KeysResponse, error) {
	return nil, nil
}

func (s *Server) FullScanOwnedNotes(ctx context.Context, req *rusk.ViewKey) (*rusk.OwnedNotesResponse, error) {
	return nil, nil
}

func (s *Server) NewTransaction(ctx context.Context, req *rusk.NewTransactionRequest) (*rusk.Transaction, error) {
	return nil, nil
}

func (s *Server) GetBalance(ctx context.Context, req *rusk.GetBalanceRequest) (*rusk.GetBalanceResponse, error) {
	return nil, nil
}

func (s *Server) VerifyTransaction(ctx context.Context, req *rusk.ContractCallTx) (*rusk.VerifyTransactionResponse, error) {
	return nil, nil
}

func (s *Server) CalculateMempoolBalance(ctx context.Context, req *rusk.CalculateMempoolBalanceRequest) (*rusk.GetBalanceResponse, error) {
	return nil, nil
}

func (s *Server) NewStake(ctx context.Context, req *rusk.StakeTransactionRequest) (*rusk.StakeTransaction, error) {
	return nil, nil
}

func (s *Server) VerifyStake(ctx context.Context, req *rusk.StakeTransaction) (*rusk.VerifyTransactionResponse, error) {
	return nil, nil
}

func (s *Server) NewWithdrawStake(ctx context.Context, req *rusk.WithdrawStakeTransactionRequest) (*rusk.WithdrawStakeTransaction, error) {
	return nil, nil
}

func (s *Server) NewBid(ctx context.Context, req *rusk.BidTransactionRequest) (*rusk.BidTransaction, error) {
	return nil, nil
}

func (s *Server) NewWithdrawBid(ctx context.Context, req *rusk.WithdrawBidTransactionRequest) (*rusk.WithdrawBidTransaction, error) {
	return nil, nil
}

func (s *Server) NewWithdrawFees(ctx context.Context, req *rusk.WithdrawFeesTransactionRequest) (*rusk.WithdrawFeesTransaction, error) {
	return nil, nil
}

func (s *Server) NewSlash(ctx context.Context, req *rusk.SlashTransactionRequest) (*rusk.SlashTransaction, error) {
	return nil, nil
}

package ruskmock

import (
	"context"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/grpc"
)

// Config contains a list of settings that determine the behavior
// of the `Server`.
type Config struct {
	PassScoreValidation           bool
	PassTransactionValidation     bool
	PassStateTransition           bool
	PassStateTransitionValidation bool
}

// DefaultConfig returns the default configuration for the Rusk mock server.
func DefaultConfig() *Config {
	return &Config{
		PassScoreValidation:           true,
		PassTransactionValidation:     true,
		PassStateTransition:           true,
		PassStateTransitionValidation: true,
	}
}

// Server is a stand-in Rusk server, which can be used during any kind of
// testing. Its behavior can be modified depending on the settings of the
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

// Echo the rusk server to see if it's still running.
func (s *Server) Echo(ctx context.Context, req *rusk.EchoRequest) (*rusk.EchoResponse, error) {
	return &rusk.EchoResponse{}, nil
}

// ValidateStateTransition simulates a state transition validation. The outcome is dictated
// by the server configuration.
func (s *Server) ValidateStateTransition(ctx context.Context, req *rusk.ValidateStateTransitionRequest) (*rusk.ValidateStateTransitionResponse, error) {
	return nil, nil
}

// ExecuteStateTransition simulates a state transition. The outcome is dictated by the server
// configuration.
func (s *Server) ExecuteStateTransition(ctx context.Context, req *rusk.ExecuteStateTransitionRequest) (*rusk.ExecuteStateTransitionResponse, error) {
	return nil, nil
}

// GenerateScore returns a mocked Score.
// TODO: investigate if we should instead, launch a blindbid process and do this properly.
func (s *Server) GenerateScore(ctx context.Context, req *rusk.GenerateScoreRequest) (*rusk.GenerateScoreResponse, error) {
	proof, err := crypto.RandEntropy(400)
	if err != nil {
		return nil, err
	}

	score, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}

	identity, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}

	return &rusk.GenerateScoreResponse{
		Proof:    proof,
		Score:    score,
		Seed:     req.Seed,
		Identity: identity,
	}, nil
}

// VerifyScore will return either true or false, depending on the server configuration.
func (s *Server) VerifyScore(ctx context.Context, req *rusk.VerifyScoreRequest) (*rusk.VerifyScoreResponse, error) {
	return &rusk.VerifyScoreResponse{
		Success: s.cfg.PassScoreValidation,
	}, nil
}

// GenerateSecretKey returns a set of randomly generated keys. They will contain Ristretto
// points under the hood.
func (s *Server) GenerateSecretKey(ctx context.Context, req *rusk.GenerateSecretKeyRequest) (*rusk.GenerateSecretKeyResponse, error) {
	return nil, nil
}

// Keys returns the public key for a given secret key.
func (s *Server) Keys(ctx context.Context, req *rusk.SecretKey) (*rusk.KeysResponse, error) {
	return nil, nil
}

// FullScanOwnedNotes returns the inputs belonging to the given view key.
func (s *Server) FullScanOwnedNotes(ctx context.Context, req *rusk.ViewKey) (*rusk.OwnedNotesResponse, error) {
	return nil, nil
}

// NewTransaction creates a transaction and returns it to the caller.
func (s *Server) NewTransaction(ctx context.Context, req *rusk.NewTransactionRequest) (*rusk.Transaction, error) {
	return nil, nil
}

// GetBalance calculates and returns the balance of the caller.
func (s *Server) GetBalance(ctx context.Context, req *rusk.GetBalanceRequest) (*rusk.GetBalanceResponse, error) {
	return nil, nil
}

// VerifyTransaction will return true or false, depending on the server configuration.
func (s *Server) VerifyTransaction(ctx context.Context, req *rusk.ContractCallTx) (*rusk.VerifyTransactionResponse, error) {
	return &rusk.VerifyTransactionResponse{
		Verified: s.cfg.PassTransactionValidation,
	}, nil
}

// CalculateMempoolBalance will return the amount of DUSK that is pending in the mempool
// for the caller.
func (s *Server) CalculateMempoolBalance(ctx context.Context, req *rusk.CalculateMempoolBalanceRequest) (*rusk.GetBalanceResponse, error) {
	return nil, nil
}

// NewStake creates a staking transaction and returns it to the caller.
func (s *Server) NewStake(ctx context.Context, req *rusk.StakeTransactionRequest) (*rusk.StakeTransaction, error) {
	return nil, nil
}

// VerifyStake will verify a staking transaction.
// TODO: is this method really necessary?
func (s *Server) VerifyStake(ctx context.Context, req *rusk.StakeTransaction) (*rusk.VerifyTransactionResponse, error) {
	return nil, nil
}

// NewWithdrawStake creates a stake withdrawal transaction and returns it to the caller.
func (s *Server) NewWithdrawStake(ctx context.Context, req *rusk.WithdrawStakeTransactionRequest) (*rusk.WithdrawStakeTransaction, error) {
	return nil, nil
}

// NewBid creates a bidding transaction and returns it to the caller.
func (s *Server) NewBid(ctx context.Context, req *rusk.BidTransactionRequest) (*rusk.BidTransaction, error) {
	return nil, nil
}

// NewWithdrawBid creates a bid withdrawal transaction and returns it to the caller.
func (s *Server) NewWithdrawBid(ctx context.Context, req *rusk.WithdrawBidTransactionRequest) (*rusk.WithdrawBidTransaction, error) {
	return nil, nil
}

// NewWithdrawFees creates a fee withdrawal transaction and returns it to the caller.
func (s *Server) NewWithdrawFees(ctx context.Context, req *rusk.WithdrawFeesTransactionRequest) (*rusk.WithdrawFeesTransaction, error) {
	return nil, nil
}

// NewSlash creates a slashing transaction and returns it to the caller.
func (s *Server) NewSlash(ctx context.Context, req *rusk.SlashTransactionRequest) (*rusk.SlashTransaction, error) {
	return nil, nil
}

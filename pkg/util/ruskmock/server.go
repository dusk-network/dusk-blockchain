// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package ruskmock

import (
	"context"
	"math/big"
	"net"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	corewallet "github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/util/legacy"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/database"
	"github.com/dusk-network/dusk-wallet/v2/key"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
	"github.com/dusk-network/dusk-wallet/v2/wallet"
	zkproof "github.com/dusk-network/dusk-zkproof"
	"github.com/sirupsen/logrus"
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
	s   *grpc.Server

	w *wallet.Wallet
	p *user.Provisioners
}

// New returns a new Rusk mock server with the given config. If no config is
// passed, a default one is put into place.
func New(cfg *Config, c config.Registry) (*Server, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	srv := &Server{
		cfg: cfg,
		p:   user.NewProvisioners(),
	}

	grpcServer := grpc.NewServer()
	rusk.RegisterStateServer(grpcServer, srv)
	rusk.RegisterKeysServer(grpcServer, srv)
	rusk.RegisterBlindBidServiceServer(grpcServer, srv)
	rusk.RegisterBidServiceServer(grpcServer, srv)
	rusk.RegisterTransferServer(grpcServer, srv)
	rusk.RegisterStakeServiceServer(grpcServer, srv)
	srv.s = grpcServer

	// First load the database
	db, err := database.New(c.Wallet.Store + "_2")
	if err != nil {
		return nil, err
	}

	// Then load the wallet
	w, err := wallet.LoadFromFile(byte(2), db, fetchDecoys, fetchInputs, "password", c.Wallet.File)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	if err = w.UpdateWalletHeight(0); err != nil {
		return nil, err
	}

	srv.w = w

	var genesis *block.Block
	if c.Genesis.Legacy {
		// Sync up the provisioners
		genesis = legacy.DecodeGenesis()
		// Note that we don't use `chain.addConsensusNodes` here because the transaction types
		// are incompatible.
		if err = srv.addConsensusNodes(genesis.Txs, 0); err != nil {
			return nil, err
		}
	} else {
		g := config.DecodeGenesis()

		if err = chain.ReconstructCommittee(srv.p, g); err != nil {
			return nil, err
		}

		genesis, err = legacy.NewBlockToOldBlock(g)
		if err != nil {
			return nil, err
		}
	}

	if _, _, err := srv.w.CheckWireBlock(*genesis); err != nil {
		return nil, err
	}

	return srv, nil
}

// Serve will start listening on a hardcoded IP and port. The server will then accept
// incoming gRPC requests.
func (s *Server) Serve(network, url string) error {
	l, err := net.Listen(network, url)
	if err != nil {
		return err
	}

	go func() {
		if err := s.s.Serve(l); err != nil {
			logrus.WithError(err).Errorln("rusk mock server encountered an error")
		}
	}()

	return nil
}

// Echo the rusk server to see if it's still running.
func (s *Server) Echo(ctx context.Context, req *rusk.EchoRequest) (*rusk.EchoResponse, error) {
	return &rusk.EchoResponse{}, nil
}

// VerifyStateTransition simulates a state transition validation. The outcome is dictated
// by the server configuration.
func (s *Server) VerifyStateTransition(ctx context.Context, req *rusk.VerifyStateTransitionRequest) (*rusk.VerifyStateTransitionResponse, error) {
	if !s.cfg.PassStateTransitionValidation {
		indices := make([]uint64, len(req.Txs))
		for i := range indices {
			indices[i] = uint64(i)
		}

		return &rusk.VerifyStateTransitionResponse{
			FailedCalls: indices,
		}, nil
	}

	return &rusk.VerifyStateTransitionResponse{
		FailedCalls: make([]uint64, 0),
	}, nil
}

// ExecuteStateTransition simulates a state transition. The outcome is dictated by the server
// configuration.
func (s *Server) ExecuteStateTransition(ctx context.Context, req *rusk.ExecuteStateTransitionRequest) (*rusk.ExecuteStateTransitionResponse, error) {
	txs, err := legacy.ContractCallsToTxs(req.Txs)
	if err != nil {
		return nil, err
	}
	blk := block.NewBlock()
	blk.Txs = txs
	blk.Header.Height = req.Height

	_, _, err = s.w.CheckWireBlock(*blk)
	if err != nil {
		return nil, err
	}

	if err := s.addConsensusNodes(blk.Txs, req.Height); err != nil {
		return nil, err
	}

	return &rusk.ExecuteStateTransitionResponse{
		Success: s.cfg.PassStateTransition,
	}, nil
}

// GetProvisioners returns the current set of provisioners.
func (s *Server) GetProvisioners(ctx context.Context, req *rusk.GetProvisionersRequest) (*rusk.GetProvisionersResponse, error) {
	return &rusk.GetProvisionersResponse{
		Provisioners: legacy.ProvisionersToRuskCommittee(s.p),
	}, nil
}

func (s *Server) addConsensusNodes(txs []transactions.Transaction, startHeight uint64) error {
	for _, tx := range txs {
		if tx.Type() == transactions.StakeType {
			stake := tx.(*transactions.Stake)
			if err := s.p.Add(stake.PubKeyBLS, stake.Outputs[0].EncryptedAmount.BigInt().Uint64(), startHeight, startHeight+stake.Lock-2); err != nil {
				return err
			}
		}
	}

	return nil
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
		BlindbidProof: &rusk.Proof{
			Data: proof,
		},
		Score: &rusk.BlsScalar{
			Data: score,
		},
		ProverIdentity: &rusk.BlsScalar{
			Data: identity,
		},
	}, nil
}

// VerifyScore will return either true or false, depending on the server configuration.
func (s *Server) VerifyScore(ctx context.Context, req *rusk.VerifyScoreRequest) (*rusk.VerifyScoreResponse, error) {
	return &rusk.VerifyScoreResponse{
		Success: s.cfg.PassScoreValidation,
	}, nil
}

// GenerateKeys returns a set of randomly generated keys. They will contain Ristretto
// points under the hood.
func (s *Server) GenerateKeys(ctx context.Context, req *rusk.GenerateKeysRequest) (*rusk.GenerateKeysResponse, error) {
	db, err := database.New(config.Get().Wallet.Store)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = db.Close()
	}()

	seed, err := corewallet.GenerateNewSeed(nil)
	if err != nil {
		return nil, err
	}

	w, err := wallet.LoadFromSeed(seed, byte(2), db, fetchDecoys, fetchInputs, "password", config.Get().Wallet.File)
	if err != nil {
		return nil, err
	}

	s.w = w

	var r ristretto.Scalar
	r.Rand()
	pk := w.PublicKey()
	addr := pk.StealthAddress(r, 0)
	pSpend, err := w.PrivateSpend()
	if err != nil {
		return nil, err
	}

	return &rusk.GenerateKeysResponse{
		Sk: &rusk.SecretKey{
			A: &rusk.JubJubScalar{
				Data: pSpend,
			},
			B: &rusk.JubJubScalar{
				Data: make([]byte, 32),
			},
		},
		Vk: &rusk.ViewKey{
			A: &rusk.JubJubScalar{
				Data: make([]byte, 32),
			},
			BG: &rusk.JubJubCompressed{
				Data: make([]byte, 32),
			},
		},
		Pk: &rusk.PublicKey{
			AG: &rusk.JubJubCompressed{
				Data: addr.P.Bytes(),
			},
			BG: &rusk.JubJubCompressed{
				Data: make([]byte, 32),
			},
		},
	}, nil
}

// GenerateStealthAddress returns a stealth address generated from a public key.
func (s *Server) GenerateStealthAddress(ctx context.Context, req *rusk.PublicKey) (*rusk.StealthAddress, error) {
	var r ristretto.Scalar
	r.Rand()
	pk := s.w.PublicKey()
	addr := pk.StealthAddress(r, 0)
	return &rusk.StealthAddress{
		RG: &rusk.JubJubCompressed{
			Data: addr.P.Bytes(),
		},
		PkR: &rusk.JubJubCompressed{
			Data: make([]byte, 0),
		},
	}, nil
}

// NewTransfer creates a transaction and returns it to the caller.
func (s *Server) NewTransfer(ctx context.Context, req *rusk.TransferTransactionRequest) (*rusk.Transaction, error) {
	tx, err := transactions.NewStandard(0, byte(2), int64(100))
	if err != nil {
		return nil, err
	}

	var spend ristretto.Point
	_ = spend.UnmarshalBinary(req.Recipient.RG.Data)
	var view ristretto.Point
	_ = view.UnmarshalBinary(req.Recipient.PkR.Data)
	sp := key.PublicSpend(spend)
	v := key.PublicView(view)

	pk := &key.PublicKey{
		PubSpend: &sp,
		PubView:  &v,
	}

	addr, err := pk.PublicAddress(byte(2))
	if err != nil {
		return nil, err
	}

	var value ristretto.Scalar
	value.SetBigInt(big.NewInt(int64(req.Value)))
	if err := tx.AddOutput(*addr, value); err != nil {
		return nil, err
	}

	if err := s.w.Sign(tx); err != nil {
		return nil, err
	}

	return legacy.TxToRuskTx(tx)
}

// NewStake creates a staking transaction and returns it to the caller.
func (s *Server) NewStake(ctx context.Context, req *rusk.StakeTransactionRequest) (*rusk.Transaction, error) {
	stake, err := transactions.NewStake(0, byte(2), int64(0), 250000, s.w.ConsensusKeys().EdPubKeyBytes, req.PublicKeyBls)
	if err != nil {
		return nil, err
	}

	if err := s.w.Sign(stake); err != nil {
		return nil, err
	}

	return legacy.StakeToRuskStake(stake)
}

// NewBid creates a bidding transaction and returns it to the caller.
func (s *Server) NewBid(ctx context.Context, req *rusk.BidTransactionRequest) (*rusk.BidTransaction, error) {
	var k ristretto.Scalar
	_ = k.UnmarshalBinary(req.K.Data)
	m := zkproof.CalculateM(k)
	bid, err := transactions.NewBid(0, byte(2), int64(0), 250000, m.Bytes())
	if err != nil {
		return nil, err
	}

	if err := s.w.Sign(bid); err != nil {
		return nil, err
	}

	return legacy.BidToRuskBid(bid)
}

// FindBid will return all of the bids for a given stealth address.
// TODO: implement
func (s *Server) FindBid(ctx context.Context, req *rusk.FindBidRequest) (*rusk.BidList, error) {
	return nil, nil
}

func fetchInputs(netPrefix byte, db *database.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error) {
	// Fetch all inputs from database that are >= totalAmount
	// returns error if inputs do not add up to total amount
	privSpend, err := key.PrivateSpend()
	if err != nil {
		return nil, 0, err
	}
	return db.FetchInputs(privSpend.Bytes(), totalAmount)
}

// This will just mock some decoys. Note that, if we change to actual tx verification,
// this should be updated accordingly.
func fetchDecoys(numMixins int) []mlsag.PubKeys {
	var decoys []ristretto.Point
	for i := 0; i < numMixins; i++ {
		decoy := ristretto.Point{}
		decoy.Rand()

		decoys = append(decoys, decoy)
	}

	var pubKeys []mlsag.PubKeys
	for i := 0; i < numMixins; i++ {
		var keyVector mlsag.PubKeys
		keyVector.AddPubKey(decoys[i])

		var secondaryKey ristretto.Point
		secondaryKey.Rand()
		keyVector.AddPubKey(secondaryKey)

		pubKeys = append(pubKeys, keyVector)
	}
	return pubKeys
}

package candidate

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

var _ consensus.Component = (*Generator)(nil)

var lg = log.WithField("process", "candidate generator")

// MaxTxSetSize defines the maximum amount of transactions.
// It is TBD along with block size and processing.MaxFrameSize
const MaxTxSetSize = 150000

// Generator is responsible for generating candidate blocks, and propagating them
// alongside received Scores. It is triggered by the ScoreEvent, sent by the score generator.
type Generator struct {
	publisher eventbus.Publisher

	genPrivKey *key.PrivateKey
	rpcBus     *rpcbus.RPCBus
	signer     consensus.Signer

	roundInfo    consensus.RoundUpdate
	scoreEventID uint32

	rusk rusk.RuskClient
}

// NewComponent returns an uninitialized candidate generator.
func NewComponent(publisher eventbus.Publisher, genPrivKey *key.PrivateKey, rpcBus *rpcbus.RPCBus, rusk rusk.RuskClient) *Generator {
	return &Generator{
		publisher:  publisher,
		rpcBus:     rpcBus,
		genPrivKey: genPrivKey,
		rusk:       rusk,
	}
}

// Initialize the Generator, by populating the fields needed to generate candidate
// blocks, and returns a Listener for ScoreEvents.
// Implements consensus.Component.
func (bg *Generator) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, ru consensus.RoundUpdate) []consensus.TopicListener {
	bg.roundInfo = ru
	bg.signer = signer

	scoreEventListener := consensus.TopicListener{
		Topic:    topics.ScoreEvent,
		Listener: consensus.NewSimpleListener(bg.Collect, consensus.LowPriority, false),
	}
	bg.scoreEventID = scoreEventListener.Listener.ID()

	return []consensus.TopicListener{scoreEventListener}
}

// ID returns the listener ID of the Generator.
// Implements consensus.Component.
func (bg *Generator) ID() uint32 {
	return bg.scoreEventID
}

// Finalize implements consensus.Component
func (bg *Generator) Finalize() {}

// ScoreFactory is the PacketFactory implementation to let the signer  scores
type ScoreFactory struct {
	sp       message.ScoreProposal
	prevHash []byte
	voteHash []byte
}

// Create a score message by setting the right header. It complies with the
// consensus.PacketFactory interface
func (sf ScoreFactory) Create(sender []byte, round uint64, step uint8) consensus.InternalPacket {
	hdr := sf.sp.State()
	if hdr.Round != round || hdr.Step != step {
		lg.Panicf("mismatch of Header round and step in score creation. ScoreProposal has a different Round and Step (%d, %d) than the Coordinator (%d, %d)", hdr.Round, hdr.Step, round, step)
	}
	score := message.NewScore(sf.sp, sender, sf.prevHash, sf.voteHash)
	return *score
}

// Collect a `ScoreProposal`, which triggers generation of a `Score` and a
// candidate `block.Block`
// The Generator will propagate both the Score and Candidate messages at the end
// of this function call.
func (bg *Generator) Collect(e consensus.InternalPacket) error {
	sev := e.(message.ScoreProposal)

	blk, err := bg.Generate(sev)
	if err != nil {
		return err
	}

	scoreFactory := ScoreFactory{sev, bg.roundInfo.Hash, blk.Header.Hash}
	score := bg.signer.Compose(scoreFactory)
	lg.Debugln("sending score")
	msg := message.New(topics.Score, score)
	if e := bg.signer.Gossip(msg, bg.ID()); e != nil {
		return e
	}

	// Create candidate message
	resp, err := bg.rpcBus.Call(topics.GetLastCertificate, rpcbus.EmptyRequest(), 5*time.Second)
	if err != nil {
		return err
	}
	certBuf := resp.(bytes.Buffer)

	cert := block.EmptyCertificate()
	if err := message.UnmarshalCertificate(&certBuf, cert); err != nil {
		return err
	}

	// Since the Candidate message goes straight to the Chain, there is
	// no need to use `SendAuthenticated`, as the header is irrelevant.
	// Thus, we will instead gossip it directly.
	lg.Debugln("sending candidate")
	candidateMsg := message.MakeCandidate(blk, cert)
	msg = message.New(topics.Candidate, candidateMsg)
	return bg.signer.Gossip(msg, bg.ID())
}

// Generate a Block
func (bg *Generator) Generate(sev message.ScoreProposal) (*block.Block, error) {
	return bg.GenerateBlock(bg.roundInfo.Round, sev.Seed, sev.Proof, sev.Score, bg.roundInfo.Hash)
}

// GenerateBlock generates a candidate block, by constructing the header and filling it
// with transactions from the mempool.
func (bg *Generator) GenerateBlock(round uint64, seed, proof, score, prevBlockHash []byte) (*block.Block, error) {
	txs, err := bg.ConstructBlockTxs(proof, score)
	if err != nil {
		return nil, err
	}

	// Construct header
	h := &block.Header{
		Version:       0,
		Timestamp:     time.Now().Unix(),
		Height:        round,
		PrevBlockHash: prevBlockHash,
		TxRoot:        nil,
		Seed:          seed,
		Certificate:   block.EmptyCertificate(),
	}

	// Construct the candidate block
	candidateBlock := &block.Block{
		Header: h,
		Txs:    txs,
	}

	// Update TxRoot
	root, err := candidateBlock.CalculateRoot()
	if err != nil {
		return nil, err
	}
	candidateBlock.Header.TxRoot = root

	// Generate the block hash
	hash, err := candidateBlock.CalculateHash()
	if err != nil {
		return nil, err
	}
	candidateBlock.Header.Hash = hash

	return candidateBlock, nil
}

// ConstructBlockTxs will fetch all valid transactions from the mempool, append a coinbase
// transaction, and return them all.
func (bg *Generator) ConstructBlockTxs(proof, score []byte) ([]transactions.ContractCall, error) {
	txs := make([]transactions.ContractCall, 0)

	// Retrieve and append the verified transactions from Mempool
	if bg.rpcBus != nil {

		// Max transaction size param
		param := new(bytes.Buffer)
		if err := encoding.WriteUint32LE(param, uint32(MaxTxSetSize)); err != nil {
			return nil, err
		}

		resp, err := bg.rpcBus.Call(topics.GetMempoolTxsBySize, rpcbus.NewRequest(*param), 4*time.Second)
		// TODO: GetVerifiedTxs should ensure once again that none of the txs have been
		// already accepted in the chain.
		if err != nil {
			return nil, err
		}
		txs = append(txs, resp.([]transactions.ContractCall)...)
	}

	// Construct and append coinbase Tx to reward the generator
	coinbaseTx, err := bg.constructCoinbaseTx()
	if err != nil {
		return nil, err
	}

	txs = append(txs, coinbaseTx)

	return txs, nil
}

// ConstructCoinbaseTx forges the transaction to reward the block generator.
func (bg *Generator) constructCoinbaseTx() (*transactions.DistributeTransaction, error) {
	req := &rusk.DistributeTransactionRequest{
		Tx: &rusk.NewTransactionRequest{
			Sk:         &rusk.SecretKey{}, // TODO: get from genPrivKey
			Recipient:  &rusk.PublicKey{}, // TODO: bg.privKey.PublicKey()
			Value:      config.GeneratorReward,
			Fee:        100, // TODO: what do we set as fee here?
			Obfuscated: false,
		},
	}

	resp, err := bg.rusk.NewDistribute(context.Background(), req)
	if err != nil {
		return nil, err
	}

	tx := &transactions.DistributeTransaction{}
	if err := transactions.UDistribute(resp, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

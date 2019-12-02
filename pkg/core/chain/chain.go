package chain

import (
	"bytes"

	"encoding/binary"
	"fmt"
	"sync"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/block"
	zkproof "github.com/dusk-network/dusk-zkproof"
	logger "github.com/sirupsen/logrus"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/core/verifiers"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-wallet/transactions"
	"golang.org/x/crypto/ed25519"
)

var log *logger.Entry = logger.WithFields(logger.Fields{"process": "chain"})

// Chain represents the nodes blockchain
// This struct will be aware of the current state of the node.
type Chain struct {
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	db       database.DB
	p        *user.Provisioners
	bidList  *user.BidList

	prevBlock block.Block
	// protect prevBlock with mutex as it's touched out of the main chain loop
	// by SubscribeCallback.
	// TODO: Consider if mutex can be removed
	mu sync.RWMutex

	// collector channels
	candidateChan <-chan *block.Block

	// rpcbus channels
	getLastBlockChan         <-chan rpcbus.Request
	verifyCandidateBlockChan <-chan rpcbus.Request
	getCandidateChan         <-chan rpcbus.Request
}

// New returns a new chain object
func New(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus) (*Chain, error) {
	_, db := heavy.CreateDBConnection()

	l, err := newLoader(db)
	if err != nil {
		return nil, fmt.Errorf("%s on loading chain db '%s'", err.Error(), cfg.Get().Database.Dir)
	}

	// set up collectors
	candidateChan := initCandidateCollector(eventBus)

	// set up rpcbus channels
	getLastBlockChan := make(chan rpcbus.Request, 1)
	verifyCandidateBlockChan := make(chan rpcbus.Request, 1)
	getCandidateChan := make(chan rpcbus.Request, 1)
	rpcBus.Register(rpcbus.GetLastBlock, getLastBlockChan)
	rpcBus.Register(rpcbus.VerifyCandidateBlock, verifyCandidateBlockChan)
	rpcBus.Register(rpcbus.GetCandidate, getCandidateChan)

	chain := &Chain{
		eventBus:                 eventBus,
		rpcBus:                   rpcBus,
		db:                       db,
		prevBlock:                *l.chainTip,
		candidateChan:            candidateChan,
		p:                        user.NewProvisioners(),
		bidList:                  &user.BidList{},
		getLastBlockChan:         getLastBlockChan,
		verifyCandidateBlockChan: verifyCandidateBlockChan,
		getCandidateChan:         getCandidateChan,
	}

	chain.restoreConsensusData()

	// Hook the chain up to the required topics
	cbListener := eventbus.NewCallbackListener(chain.onAcceptBlock)
	eventBus.Subscribe(topics.Block, cbListener)
	eventBus.Register(topics.Candidate, consensus.NewRepublisher(eventBus, topics.Candidate), &consensus.Validator{})
	return chain, nil
}

// Listen to the collectors
func (c *Chain) Listen() {
	for {
		select {

		case b := <-c.candidateChan:
			_ = c.handleCandidateBlock(*b)
		// wire.RPCBus requests handlers
		case r := <-c.getLastBlockChan:

			buf := new(bytes.Buffer)

			c.mu.RLock()
			prevBlock := c.prevBlock
			c.mu.RUnlock()

			if err := marshalling.MarshalBlock(buf, &prevBlock); err != nil {
				r.RespChan <- rpcbus.Response{bytes.Buffer{}, err}
				continue
			}

			r.RespChan <- rpcbus.Response{*buf, nil}
		case r := <-c.verifyCandidateBlockChan:
			if err := c.verifyCandidateBlock(r.Params.Bytes()); err != nil {
				r.RespChan <- rpcbus.Response{bytes.Buffer{}, err}
				continue
			}

			r.RespChan <- rpcbus.Response{bytes.Buffer{}, nil}
		case r := <-c.getCandidateChan:
			blk, err := c.fetchCandidateBlock(r.Params.Bytes())
			if err != nil {
				r.RespChan <- rpcbus.Response{bytes.Buffer{}, err}
				continue
			}

			buf := new(bytes.Buffer)
			if err := marshalling.MarshalBlock(buf, blk); err != nil {
				r.RespChan <- rpcbus.Response{bytes.Buffer{}, err}
				continue
			}

			r.RespChan <- rpcbus.Response{*buf, nil}
		}
	}
}

// LaunchConsensus will listen for an init message, and send a round update
// once it is received.
func (c *Chain) LaunchConsensus() {
	initChan := make(chan bytes.Buffer, 1)
	l := eventbus.NewChanListener(initChan)
	id := c.eventBus.Subscribe(topics.Initialization, l)
	<-initChan
	c.mu.RLock()
	defer c.mu.RUnlock()
	// height + 2 is the new round, since blocks are finalized after
	// 2 rounds.
	c.sendRoundUpdate(c.prevBlock.Header.Height + 2)
	c.eventBus.Unsubscribe(topics.Initialization, id)
}

func (c *Chain) propagateBlock(blk block.Block) error {
	buffer := topics.Block.ToBuffer()
	if err := marshalling.MarshalBlock(&buffer, &blk); err != nil {
		return err
	}

	c.eventBus.Publish(topics.Gossip, &buffer)
	return nil
}

func (c *Chain) addBidder(tx *transactions.Bid, startHeight uint64) {
	var bid user.Bid
	x := calculateXFromBytes(tx.Outputs[0].Commitment.Bytes(), tx.M)
	copy(bid.X[:], x.Bytes())
	copy(bid.M[:], tx.M)
	bid.EndHeight = startHeight + tx.Lock
	c.addBid(bid)
}

func (c *Chain) Close() error {

	log.Info("Close database")

	drvr, err := database.From(cfg.Get().Database.Driver)
	if err != nil {
		return err
	}

	return drvr.Close()
}

func (c *Chain) onAcceptBlock(m bytes.Buffer) error {
	blk := block.NewBlock()
	if err := marshalling.UnmarshalBlock(&m, blk); err != nil {
		return err
	}

	return c.AcceptBlock(*blk)
}

// AcceptBlock will accept a block if
// 1. We have not seen it before
// 2. All stateless and statefull checks are true
// Returns nil, if checks passed and block was successfully saved
func (c *Chain) AcceptBlock(blk block.Block) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	field := logger.Fields{"process": "accept block"}
	l := log.WithFields(field)

	l.Trace("verifying block")

	// 1. Check that stateless and stateful checks pass
	if err := verifiers.CheckBlock(c.db, c.prevBlock, blk); err != nil {
		l.WithError(err).Warnln("block verification failed")
		return err
	}

	// 2. Check the certificate
	// This check should avoid a possible race condition between accepting two blocks
	// at the same height, as the probability of the committee creating two valid certificates
	// for the same round is negligible.
	l.Trace("verifying block certificate")
	if err := verifiers.CheckBlockCertificate(*c.p, blk); err != nil {
		l.WithError(err).Warnln("certificate verification failed")
		return err
	}

	// 3. Add provisioners and block generators
	l.Trace("adding consensus nodes")
	c.addConsensusNodes(blk.Txs, blk.Header.Height)

	// 4. Store block in database
	l.Trace("storing block in db")
	err := c.db.Update(func(t database.Transaction) error {
		return t.StoreBlock(&blk)
	})

	if err != nil {
		l.WithError(err).Errorln("block storing failed")
		return err
	}

	c.prevBlock = blk

	// 5. Gossip advertise block Hash
	l.Trace("gossiping block")
	if err := c.advertiseBlock(blk); err != nil {
		l.WithError(err).Errorln("block advertising failed")
		return err
	}

	// 6. Cleanup obsolete candidate blocks
	l.Trace("cleaning obsolete candidate blocks")
	var count uint32
	err = c.db.Update(func(t database.Transaction) error {
		count, err = t.DeleteCandidateBlocks(blk.Header.Height)
		return err
	})

	if err != nil {
		// Not critical enough to abort the accepting procedure
		log.WithError(err).Warnln("deleting candidate blocks failed")
	} else {
		log.WithField("count", count).Traceln("candidate blocks deleted")
	}

	// 7. Remove expired provisioners and bids
	// We remove provisioners and bids from accepted block height + 1,
	// to set up our committee correctly for the next block.
	l.Trace("removing expired consensus transactions")
	c.removeExpiredProvisioners(blk.Header.Height + 2)
	c.removeExpiredBids(blk.Header.Height + 2)

	// 8. Notify other subsystems for the accepted block
	// Subsystems listening for this topic:
	// mempool.Mempool
	// consensus.generation.broker
	l.Trace("notifying internally")
	buf := new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf, &blk); err != nil {
		l.WithError(err).Errorln("block encoding failed")
		return err
	}

	c.eventBus.Publish(topics.AcceptedBlock, buf)

	// 9. Send round update
	// We send a round update after accepting a new block, which should include
	// a set of provisioners, and a bidlist. This allows the consensus components
	// to rehydrate their state properly for the next round.
	l.Trace("sending round update")
	if err := c.sendRoundUpdate(blk.Header.Height + 2); err != nil {
		l.WithError(err).Errorln("sending round update failed")
		return err
	}

	l.Trace("procedure ended")
	return nil
}

func (c *Chain) sendRoundUpdate(round uint64) error {
	buf := new(bytes.Buffer)
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, round)
	if _, err := buf.Write(roundBytes); err != nil {
		return err
	}
	membersBuf, err := c.marshalProvisioners()
	if err != nil {
		return err
	}

	if _, err := buf.ReadFrom(membersBuf); err != nil {
		return err
	}

	if err := user.MarshalBidList(buf, *c.bidList); err != nil {
		return err
	}

	c.eventBus.Publish(topics.RoundUpdate, buf)
	return nil
}

func (c *Chain) addConsensusNodes(txs []transactions.Transaction, startHeight uint64) {
	field := logger.Fields{"process": "accept block"}
	l := log.WithFields(field)

	for _, tx := range txs {
		switch tx.Type() {
		case transactions.StakeType:
			stake := tx.(*transactions.Stake)
			if err := c.addProvisioner(stake.PubKeyEd, stake.PubKeyBLS, stake.Outputs[0].EncryptedAmount.BigInt().Uint64(), startHeight+stake.Lock); err != nil {
				l.Errorf("adding provisioner failed: %s", err.Error())
			}
		case transactions.BidType:
			bid := tx.(*transactions.Bid)
			c.addBidder(bid, startHeight)
		}
	}
}

func (c *Chain) handleCandidateBlock(candidate block.Block) error {
	// Save it into persistent storage
	err := c.db.Update(func(t database.Transaction) error {
		return t.StoreCandidateBlock(&candidate)
	})

	if err != nil {
		log.Errorf("storing the candidate block failed: %s", err.Error())
		return err
	}

	return nil
}

func (c *Chain) fetchCandidateBlock(hash []byte) (*block.Block, error) {
	var candidate *block.Block
	err := c.db.View(func(t database.Transaction) error {
		var err error
		candidate, err = t.FetchCandidateBlock(hash)
		return err
	})

	return candidate, err
}

func (c *Chain) verifyCandidateBlock(hash []byte) error {
	candidate, err := c.fetchCandidateBlock(hash)
	if err != nil {
		return err
	}

	return verifiers.CheckBlock(c.db, c.prevBlock, *candidate)
}

// Send Inventory message to all peers
func (c *Chain) advertiseBlock(b block.Block) error {
	msg := &peermsg.Inv{}
	msg.AddItem(peermsg.InvTypeBlock, b.Header.Hash)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		panic(err)
	}

	if err := topics.Prepend(buf, topics.Inv); err != nil {
		return err
	}

	c.eventBus.Publish(topics.Gossip, buf)
	return nil
}

// TODO: consensus data should be persisted to disk, to decrease
// startup times
func (c *Chain) restoreConsensusData() {
	var currentHeight uint64
	err := c.db.View(func(t database.Transaction) error {
		var err error
		currentHeight, err = t.FetchCurrentHeight()
		return err
	})

	if err != nil {
		currentHeight = 0
	}

	searchingHeight := uint64(0)
	if currentHeight > transactions.MaxLockTime {
		searchingHeight = currentHeight - transactions.MaxLockTime
	}

	for {
		var blk *block.Block
		err := c.db.View(func(t database.Transaction) error {
			hash, err := t.FetchBlockHashByHeight(searchingHeight)
			if err != nil {
				return err
			}

			blk, err = t.FetchBlock(hash)
			return err
		})

		if err != nil {
			break
		}

		for _, tx := range blk.Txs {
			switch t := tx.(type) {
			case *transactions.Stake:
				// Only add them if their stake is still valid
				if searchingHeight+t.Lock > currentHeight {
					amount := t.Outputs[0].EncryptedAmount.BigInt().Uint64()
					c.addProvisioner(t.PubKeyEd, t.PubKeyBLS, amount, searchingHeight+t.Lock)
				}
			case *transactions.Bid:
				// TODO: The commitment to D is turned (in quite awful fashion) from a Point into a Scalar here,
				// to work with the `zkproof` package. Investigate if we should change this (reserve for testnet v2,
				// as this is most likely a consensus-breaking change)
				if searchingHeight+t.Lock > currentHeight {
					c.addBidder(t, searchingHeight)
				}
			}
		}

		searchingHeight++
	}
	return
}

// RemoveExpired removes Provisioners which stake expired
func (c *Chain) removeExpiredProvisioners(round uint64) {
	for pk, member := range c.p.Members {
		for i := 0; i < len(member.Stakes); i++ {
			if member.Stakes[i].EndHeight < round {
				member.RemoveStake(i)
				// If they have no stakes left, we should remove them entirely.
				if len(member.Stakes) == 0 {
					c.removeProvisioner([]byte(pk))
				}

				// Reset index
				i = -1
			}
		}
	}
}

// addProvisioner will add a Member to the Provisioners by using the bytes of a BLS public key.
func (c *Chain) addProvisioner(pubKeyEd, pubKeyBLS []byte, amount, endHeight uint64) error {
	if len(pubKeyEd) != 32 {
		return fmt.Errorf("public key is %v bytes long instead of 32", len(pubKeyEd))
	}

	if len(pubKeyBLS) != 129 {
		return fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	i := string(pubKeyBLS)
	stake := user.Stake{amount, endHeight}

	// Check for duplicates
	_, inserted := c.p.Set.IndexOf(pubKeyBLS)
	if inserted {
		// If they already exist, just add their new stake
		c.p.Members[i].AddStake(stake)
		return nil
	}

	// This is a new provisioner, so let's initialize the Member struct and add them to the list
	c.p.Set.Insert(pubKeyBLS)
	m := &user.Member{}
	m.PublicKeyEd = ed25519.PublicKey(pubKeyEd)

	m.PublicKeyBLS = pubKeyBLS
	m.AddStake(stake)

	c.p.Members[i] = m
	return nil
}

// Remove a Member, designated by their BLS public key.
func (c *Chain) removeProvisioner(pubKeyBLS []byte) bool {
	delete(c.p.Members, string(pubKeyBLS))
	return c.p.Set.Remove(pubKeyBLS)
}

func (c *Chain) marshalProvisioners() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := user.MarshalProvisioners(buf, c.p)
	return buf, err
}

func calculateXFromBytes(d, m []byte) ristretto.Scalar {
	var dBytes [32]byte
	copy(dBytes[:], d)
	var mBytes [32]byte
	copy(mBytes[:], m)
	var dScalar ristretto.Scalar
	dScalar.SetBytes(&dBytes)
	var mScalar ristretto.Scalar
	mScalar.SetBytes(&mBytes)
	x := zkproof.CalculateX(dScalar, mScalar)
	return x
}

// AddBid will add a bid to the BidList.
func (c *Chain) addBid(bid user.Bid) {
	// Check for duplicates
	for _, bidFromList := range *c.bidList {
		if bidFromList.Equals(bid) {
			return
		}
	}

	*c.bidList = append(*c.bidList, bid)
}

// RemoveBid will iterate over a BidList to remove a specified bid.
func (c *Chain) removeBid(bid user.Bid) {
	for i, bidFromList := range *c.bidList {
		if bidFromList.Equals(bid) {
			c.bidList.Remove(i)
		}
	}
}

// RemoveExpired iterates over a BidList to remove expired bids.
func (c *Chain) removeExpiredBids(round uint64) {
	for _, bid := range *c.bidList {
		if bid.EndHeight < round {
			// We need to call RemoveBid here and loop twice, as the index
			// could be off if more than one bid is removed.
			c.removeBid(bid)
		}
	}
}

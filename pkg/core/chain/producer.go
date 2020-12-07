package chain

import (
	"bytes"
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
)

type Producer struct {
	*Chain

	// Consensus loop
	loop           *loop.Consensus
	pubKey         *keys.PublicKey
	requestor      *candidate.Requestor
	newBlockChan   chan consensus.Results
	CatchBlockChan chan consensus.Results
}

func New(chain *Chain, loop *loop.Consensus, pubKey *keys.PublicKey, requestor *candidate.Requestor) *Producer {

}

// ProduceBlocks will
func (p *Producer) ProduceBlocks(ctx context.Context) error {
	for {
		candidate := p.crunchBlock(ctx)
		block, err := candidate.Blk, candidate.Err
		if err != nil {
			return err
		}

		// Otherwise, accept the block directly.
		if !block.IsEmpty() {
			if err = c.AppendBlock(ctx, block); err != nil {
				return err
			}
		}
	}
}

func (p *Producer) crunchBlock(ctx context.Context) (winner consensus.Results) {
	crunchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ru := p.GetRoundUpdate()

	go p.catchNewBlocks(crunchCtx, ru)

	if c.loop != nil {
		scr, agr, err := loop.CreateStateMachine(p.loop.Emitter, p.db, config.ConsensusTimeOut, p.pubKey.Copy(), p.VerifyCandidateBlock, p.requestor, p.newBlockChan)
		if err != nil {
			// TODO: errors should be handled by the caller
			log.WithError(err).Error("could not create consensus state machine")
			winner.Err = err
			return winner
		}

		return p.loop.Spin(ctx, scr, agr, ru)
	}

	return <-p.newBlockChan
}

func (p *Producer) catchNewBlocks(ctx context.Context, ru consensus.RoundUpdate) {
	for {
		select {
		case r := <-p.CatchBlockChan:
			if r.Blk.Header != nil && r.Blk.Header.Height != ru.Round {
				continue
			}

			c.newBlockChan <- r
		case <-ctx.Done():
			return
		}
	}
}

// NotifyBlock will handle blocks incoming from the network,
// which directly succeed the known chain tip.
func (p *Producer) NotifyBlock(ctx context.Context, blk block.Block) {
	log.WithField("height", blk.Header.Height).Trace("received succeeding block")

	select {
	case p.CatchBlockChan <- consensus.Results{Blk: blk, Err: nil}:
	default:
	}
}

// AppendBlock will accept a block which directly follows the chain
// tip, and advertises it to the node's peers.
func (p *Producer) AppendBlock(ctx context.Context, blk block.Block) error {
	log.WithField("height", blk.Header.Height).Trace("accepting succeeding block")
	if err := c.acceptBlock(ctx, blk); err != nil {
		return err
	}

	log.Trace("gossiping block")
	if err := p.advertiseBlock(blk); err != nil {
		log.WithError(err).Error("block advertising failed")
		return err
	}

	return nil
}

// Send Inventory message to all peers
func (p *Producer) advertiseBlock(b block.Block) error {
	// Disable gossiping messages if kadcast mode
	if config.Get().Kadcast.Enabled {
		return nil
	}

	msg := &message.Inv{}
	msg.AddItem(message.InvTypeBlock, b.Header.Hash)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		//TODO: shall this really panic ?
		log.Panic(err)
	}

	if err := topics.Prepend(buf, topics.Inv); err != nil {
		//TODO: shall this really panic ?
		log.Panic(err)
	}

	m := message.New(topics.Inv, *buf)
	errList := p.eventBus.Publish(topics.Gossip, m)
	diagnostics.LogPublishErrors("chain/chain.go, topics.Gossip, topics.Inv", errList)

	return nil
}

//nolint:unused
func (p *Producer) kadcastBlock(m message.Message) error {
	var kadHeight byte = 255
	if len(m.Header()) > 0 {
		kadHeight = m.Header()[0]
	}

	b, ok := m.Payload().(block.Block)
	if !ok {
		return errors.New("message payload not a block")
	}

	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, &b); err != nil {
		return err
	}

	if err := topics.Prepend(buf, topics.Block); err != nil {
		return err
	}

	m = message.NewWithHeader(topics.Block, *buf, []byte{kadHeight})
	c.eventBus.Publish(topics.Kadcast, m)

	return nil
}

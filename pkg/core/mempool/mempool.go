// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package mempool

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	logger "github.com/sirupsen/logrus"
)

var log = logger.WithFields(logger.Fields{"process": "mempool"})

const (
	idleTime        = 20 * time.Second
	backendHashmap  = "hashmap"
	backendDiskpool = "diskpool"
)

var (
	// ErrAlreadyExists transaction with the same txid already exists in mempool.
	ErrAlreadyExists = errors.New("already exists")
	// ErrAlreadyExistsInBlockchain transaction with the same txid already exists in blockchain.
	ErrAlreadyExistsInBlockchain = errors.New("already exists in blockchain")
	// ErrNullifierExists nullifier(s) already exists in the mempool state.
	ErrNullifierExists = errors.New("nullifier(s) already exists in the mempool")
)

// Mempool is a storage for the chain transactions that are valid according to the
// current chain state and can be included in the next block.
type Mempool struct {
	getMempoolTxsChan       <-chan rpcbus.Request
	getMempoolTxsBySizeChan <-chan rpcbus.Request
	sendTxChan              <-chan rpcbus.Request

	// verified txs to be included in next block.
	verified Pool

	pendingPropagation chan TxDesc

	// the collector to listen for new accepted blocks.
	acceptedBlockChan <-chan block.Block

	eventBus *eventbus.EventBus

	// the magic function that knows best what is valid chain Tx.
	verifier transactions.UnconfirmedTxProber

	limiter *rate.Limiter

	db database.DB
}

// NewMempool instantiates and initializes node mempool.
func NewMempool(db database.DB, eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, verifier transactions.UnconfirmedTxProber) *Mempool {
	log.Infof("create instance")

	l := log.WithField("backend_type", config.Get().Mempool.PoolType).
		WithField("max_size_mb", config.Get().Mempool.MaxSizeMB)

	getMempoolTxsChan := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(topics.GetMempoolTxs, getMempoolTxsChan); err != nil {
		log.WithError(err).Error("failed to register topics.GetMempoolTxs")
	}

	getMempoolTxsBySizeChan := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(topics.GetMempoolTxsBySize, getMempoolTxsBySizeChan); err != nil {
		log.WithError(err).Error("failed to register topics.GetMempoolTxsBySize")
	}

	sendTxChan := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(topics.SendMempoolTx, sendTxChan); err != nil {
		log.WithError(err).Error("failed to register topics.SendMempoolTx")
	}

	acceptedBlockChan, _ := consensus.InitAcceptedBlockUpdate(eventBus)

	// Enable rate limiter from config
	cfg := config.Get().Mempool

	var limiter *rate.Limiter

	if len(cfg.PropagateTimeout) > 0 {
		timeout, err := time.ParseDuration(cfg.PropagateTimeout)
		if err != nil {
			log.WithError(err).Fatal("could not parse mempool propagation timeout")
		}

		burst := cfg.PropagateBurst
		if burst == 0 {
			burst = 1
		}

		limiter = rate.NewLimiter(rate.Every(timeout), int(burst))

		l = l.WithField("propagate_timeout", cfg.PropagateTimeout).
			WithField("propagate_burst", burst)
	}

	m := &Mempool{
		eventBus:                eventBus,
		acceptedBlockChan:       acceptedBlockChan,
		getMempoolTxsChan:       getMempoolTxsChan,
		getMempoolTxsBySizeChan: getMempoolTxsBySizeChan,
		sendTxChan:              sendTxChan,
		verifier:                verifier,
		limiter:                 limiter,
		pendingPropagation:      make(chan TxDesc, 1000),
		db:                      db,
	}

	// Setting the pool where to cache verified transactions.
	// The pool is normally a Hashmap
	m.verified = m.newPool()

	l.Info("running")

	return m
}

func (m *Mempool) RequestUpdates() {
	maxNodes := byte(config.
		Get().
		Mempool.
		MaxNumUpdaters)

	if maxNodes == 0 {
		log.Warn("updates disabled")
		return
	}

	log.WithField("num_nodes", maxNodes).Info("request updates")

	msg := message.NewWithHeader(topics.MemPool, nil, []byte{maxNodes})
	m.eventBus.Publish(topics.KadcastRandomPoints, msg)
}

// Run spawns the mempool lifecycle routines.
func (m *Mempool) Run(ctx context.Context) {
	// Main Loop
	go m.Loop(ctx)

	// Loop to drain pendingPropagation and try to propagate transaction
	go m.propagateLoop(ctx)
}

// Loop listens for GetMempoolTxs request and topics.AcceptedBlock events.
func (m *Mempool) Loop(ctx context.Context) {
	ticker := time.NewTicker(idleTime)
	defer ticker.Stop()

	for {
		select {
		case r := <-m.getMempoolTxsChan:
			handleRequest(r, m.processGetMempoolTxsRequest, "GetMempoolTxs")
		case r := <-m.getMempoolTxsBySizeChan:
			handleRequest(r, m.processGetMempoolTxsBySizeRequest, "GetMempoolTxsBySize")
		case b := <-m.acceptedBlockChan:
			m.onBlock(b)
		case <-ticker.C:
			m.onIdle()
		case <-ctx.Done():
			m.OnClose()
			log.Info("main_loop terminated")
			return
		}

		ticker.Reset(idleTime)
	}
}

func (m *Mempool) propagateLoop(ctx context.Context) {
	for {
		select {
		case t := <-m.pendingPropagation:
			// Ensure we propagate at proper rate
			if m.limiter != nil {
				if err := m.limiter.Wait(ctx); err != nil {
					log.WithError(err).Error("failed to limit rate")
				}
			}

			txid, err := t.tx.CalculateHash()
			if err != nil {
				log.WithError(err).Error("failed to calc hash")
				continue
			}

			err = m.kadcastTx(t)
			if err != nil {
				log.WithField("txid", hex.EncodeToString(txid)).WithError(err).Error("failed to propagate")
			}

		// Mempool terminating
		case <-ctx.Done():
			log.Info("propagate_loop terminated")
			return
		}
	}
}

// ProcessTx processes a Transaction wire message.
func (m *Mempool) ProcessTx(srcPeerID string, msg message.Message) ([]bytes.Buffer, error) {
	maxSizeBytes := config.Get().Mempool.MaxSizeMB * 1000 * 1000
	if m.verified.Size() > maxSizeBytes {
		log.WithField("max_size_mb", maxSizeBytes).
			WithField("alloc_size", m.verified.Size()/1000).
			Warn("mempool is full, dropping transaction")
		return nil, errors.New("mempool is full, dropping transaction")
	}

	var h byte
	if len(msg.Header()) > 0 {
		h = msg.Header()[0]
	}

	t := TxDesc{
		tx:        msg.Payload().(transactions.ContractCall),
		received:  time.Now(),
		size:      uint(len(msg.Id())),
		kadHeight: h,
	}

	start := time.Now()
	txid, err := m.processTx(t)
	elapsed := time.Since(start)

	if err != nil {
		log.WithError(err).
			WithField("txid", toHex(txid)).
			WithField("txtype", t.tx.Type()).
			WithField("txsize", t.size).
			WithField("duration", elapsed.Microseconds()).
			WithField("kad_h", h).
			Error("failed to accept transaction")
	} else {
		log.WithField("txid", toHex(txid)).
			WithField("txtype", t.tx.Type()).
			WithField("txsize", t.size).
			WithField("duration", elapsed.Microseconds()).
			Trace("accepted transaction")
	}

	return nil, err
}

// processTx ensures all transaction rules are satisfied before adding the tx
// into the verified pool.
func (m *Mempool) processTx(t TxDesc) ([]byte, error) {
	var (
		hash []byte
		err  error
	)

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(config.Get().RPC.Rusk.ContractTimeout)*time.Millisecond)
	defer cancel()

	if hash, _, err = m.verifier.Preverify(ctx, t.tx); err != nil {
		return nil, err
	}

	t.tx, err = transactions.UpdateHash(t.tx, hash)
	if err != nil {
		return nil, fmt.Errorf("could not extend: %s", err.Error())
	}

	txid, err := t.tx.CalculateHash()
	if err != nil {
		return txid, fmt.Errorf("hash err: %s", err.Error())
	}

	// ensure transaction does not exist in the mempool state
	if m.verified.Contain(txid) {
		return txid, ErrAlreadyExists
	}

	// ensure nullifier does not exist in the mempool state
	err = m.containNullifier(t.tx)
	if err != nil {
		return txid, err
	}

	// ensure transaction does not exist in blockchain
	err = m.db.View(func(t database.Transaction) error {
		_, _, _, err = t.FetchBlockTxByHash(txid)
		return err
	})

	switch err {
	case database.ErrTxNotFound:
		t.verified = time.Now()

		// store transaction in mempool
		if err = m.verified.Put(t); err != nil {
			return txid, fmt.Errorf("store err - %v", err)
		}

		// queue transaction for (re)propagation
		go func() {
			m.pendingPropagation <- t
		}()

		return txid, nil
	case nil:
		return txid, ErrAlreadyExistsInBlockchain
	default:
		return txid, err
	}
}

// onBlock performs post-block-acceptance procedure to update mempool state
// accordingly.
func (m *Mempool) onBlock(b block.Block) {
	// Discard transactions that are accepted with this block.
	// This is the case when the accepted block has been proposed by another provisioner.
	m.discardAcceptedTxs(b.Txs)

	log.WithField("height", b.Header.Height).
		WithField("txs_count", len(b.Txs)).
		WithField("mem_alloc_size", int64(m.verified.Size())/1000).
		WithField("mem_txs_len", m.verified.Len()).
		Info("processing block completed")
}

// discardAcceptedTxs to clean up all txs from the mempool that have been already
// added to the chain.
//
// Instead of doing a full DB scan, here we rely on the latest accepted block to
// update.
//
// The passed block is supposed to be the last one accepted.
func (m *Mempool) discardAcceptedTxs(txs []transactions.ContractCall) {
	if m.verified.Len() == 0 {
		// Empty pool then no need for cleanup
		return
	}

	for _, tx := range txs {
		hash, err := tx.CalculateHash()
		if err != nil {
			log.WithError(err).Panic("could not calculate tx hash")
		}

		_ = m.verified.Delete(hash)
	}
}

func (m Mempool) containNullifier(tx transactions.ContractCall) error {
	decoded, err := tx.Decode()
	if err != nil {
		return err
	}

	if found, repeatedNullifier := m.verified.ContainAnyNullifiers(decoded.Nullifiers); found {
		h := hex.EncodeToString(repeatedNullifier)

		log.WithField("repeated_nullifier", h).Warn(ErrNullifierExists.Error())
		return ErrNullifierExists
	}

	return nil
}

// TODO: Get rid of stuck/expired transactions.
func (m *Mempool) onIdle() {
	log.
		WithField("alloc_size", int64(m.verified.Size())/1000).
		WithField("txs_count", m.verified.Len()).Info("process_on_idle")
}

func (m *Mempool) newPool() Pool {
	cfg := config.Get().Mempool

	var p Pool

	switch cfg.PoolType {
	case backendHashmap:
		p = &HashMap{
			lock:     &sync.RWMutex{},
			Capacity: cfg.HashMapPreallocTxs,
		}
	case backendDiskpool:
		p = new(buntdbPool)
	default:
		p = &HashMap{
			lock:     &sync.RWMutex{},
			Capacity: cfg.HashMapPreallocTxs,
		}
	}

	if err := p.Create(cfg.DiskPoolDir); err != nil {
		log.WithField("pool", cfg.PoolType).WithError(err).Fatal("failed to create pool")
	}

	return p
}

// processGetMempoolTxsRequest retrieves current state of the mempool of the verified but
// still unaccepted txs.
// Called by P2P on InvTypeMempoolTx msg.
func (m Mempool) processGetMempoolTxsRequest(r rpcbus.Request) (interface{}, error) {
	// Read inputs
	params := r.Params.(bytes.Buffer)
	filterTxID := params.Bytes()
	outputTxs := make([]transactions.ContractCall, 0)

	// If we are looking for a specific tx, just look it up by key.
	if len(filterTxID) == 32 {
		tx := m.verified.Get(filterTxID)
		if tx == nil {
			return outputTxs, nil
		}

		outputTxs = append(outputTxs, tx)
		return outputTxs, nil
	}

	// When filterTxID is empty, mempool returns all verified txs sorted
	// by fee from highest to lowest
	err := m.verified.RangeSort(func(k txHash, t TxDesc) (bool, error) {
		outputTxs = append(outputTxs, t.tx)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return outputTxs, err
}

// processGetMempoolTxsBySizeRequest returns a subset of verified mempool txs which
// 1. contains only highest fee txs
// 2. has total txs size not bigger than maxTxsSize (request param)
// Called by BlockGenerator on generating a new candidate block.
func (m Mempool) processGetMempoolTxsBySizeRequest(r rpcbus.Request) (interface{}, error) {
	// Read maxTxsSize param
	var maxTxsSize uint32

	params := r.Params.(bytes.Buffer)
	if err := encoding.ReadUint32LE(&params, &maxTxsSize); err != nil {
		return bytes.Buffer{}, err
	}

	txs := make([]transactions.ContractCall, 0)

	var totalSize uint32

	err := m.verified.RangeSort(func(k txHash, t TxDesc) (bool, error) {
		var done bool
		totalSize += uint32(t.size)

		if totalSize <= maxTxsSize {
			txs = append(txs, t.tx)
		} else {
			done = true
		}

		return done, nil
	})
	if err != nil {
		return bytes.Buffer{}, err
	}

	return txs, err
}

// kadcastTx (re)propagates transaction in kadcast network.
func (m *Mempool) kadcastTx(t TxDesc) error {
	if t.kadHeight > config.KadcastInitialHeight {
		return errors.New("invalid kadcast height")
	}

	/// repropagate
	buf := new(bytes.Buffer)
	if err := transactions.Marshal(buf, t.tx); err != nil {
		return err
	}

	if err := topics.Prepend(buf, topics.Tx); err != nil {
		return err
	}

	msg := message.NewWithHeader(topics.Tx, *buf, []byte{t.kadHeight})

	m.eventBus.Publish(topics.Kadcast, msg)
	return nil
}

// OnClose performs mempool cleanup procedure. It's called on canceling mempool
// context.
func (m *Mempool) OnClose() {
	// Closing diskpool backend commits changes to file and close it.
	m.verified.Close()
}

func toHex(id []byte) string {
	enc := hex.EncodeToString(id[:])
	return enc
}

// TODO: handlers should just return []transactions.ContractCall, and the
// caller should be left to format the data however they wish.
func handleRequest(r rpcbus.Request, handler func(r rpcbus.Request) (interface{}, error), name string) {
	result, err := handler(r)
	if err != nil {
		log.
			WithError(err).
			WithField("name", name).Errorf("mempool failed to process request")
		r.RespChan <- rpcbus.Response{Err: err}
		return
	}

	r.RespChan <- rpcbus.Response{Resp: result, Err: nil}
}

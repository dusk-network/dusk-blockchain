package blockchain

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	cnf "github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
)

var (
	errNoBlockchainDb    = errors.New("Blockchain database is not available")
	errInitialisedCheck  = errors.New("Failed to check if blockchain db is already initialised")
	errBlockValidation   = errors.New("Block failed sanity check")
	errBlockVerification = errors.New("Block failed to be consistent with the current blockchain")
)

// Chain holds the state of the chain
type Chain struct {
	db  *database.BlockchainDB
	net protocol.Magic
}

// New sets up a new Dusk block chain
func New(net protocol.Magic) (*Chain, error) {
	db, err := database.NewBlockchainDB(cnf.GetString("net.database.dirpath"))
	if err != nil {
		log.WithField("prefix", "blockchain").Error(err)
		return nil, errNoBlockchainDb
	}

	marker := []byte("HasBeenInitialisedAlready")
	initialised, err := db.Has(marker)
	if err != nil {
		return nil, errInitialisedCheck
	}

	//TODO: Find out if initialisation check can be done with check on existing genesis block
	if !initialised {
		// This is a new db
		log.WithField("prefix", "blockchain").Info("New Blockchain database initialisation")
		db.Put(marker, []byte{})

		// We add the genesis block into the db
		// along with the indexes for it
		if net == protocol.MainNet {

			genesisBlock, err := hex.DecodeString(GenesisBlock)
			if err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header to db")
				db.Delete(marker)
				return nil, err
			}
			r := bytes.NewReader(genesisBlock)
			b := payload.Block{}
			err = b.Decode(r)

			if err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header to db")
				db.Delete(marker)
				return nil, err
			}

			err = db.AddHeaders([]*payload.BlockHeader{b.Header})
			if err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header")
				db.Delete(marker)
				return nil, err
			}

			// No transactions in genesis block iirc

			// Cast []Payload to []Stealth
			//stealths := make([]transactions.Stealth, len(b.Txs))
			//for i, t := range b.Txs {
			//	stealths[i] = t.(transactions.Stealth)
			//}

			//err = db.AddTransactions(b.Header.Hash, stealths)
			//if err != nil {
			//	fmt.Println("Could not add Genesis Transactions")
			//	db.Delete(marker)
			//	return nil, err
			//}
		}
		if net == protocol.TestNet {
			log.WithField("prefix", "blockchain").Error("TODO: Setup the genesis block for TestNet")
			return nil, nil
		}
	}

	return &Chain{db, net}, nil
}

// AddBlock adds a block to the block chain
func (c *Chain) AddBlock(msg *payload.MsgBlock) error {
	if !validateBlock(msg) {
		return errBlockValidation
	}

	if !c.verifyBlock(msg) {
		return errBlockVerification
	}

	fmt.Println("Block Hash is ", hex.EncodeToString(msg.Block.Header.Hash))

	buf := new(bytes.Buffer)
	err := msg.Encode(buf)
	if err != nil {
		return err
	}
	return c.db.Put(msg.Block.Header.Hash, buf.Bytes())

}

// validateBlock will check the transactions,
// merkleroot is good, signature is good,every that does not require state
// This may be moved to the syncmanager. This function should not be done in a seperate go-routine
// We are intentionally blocking here because if the block is invalid, we will
// disconnect from the peer.
// We could have this return an error instead; where the error could even
// say where the validation failed, for the logs.
func validateBlock(msg *payload.MsgBlock) bool {
	return true
}

func (c *Chain) verifyBlock(msg *payload.MsgBlock) bool {
	return true
}

// ValidateHeaders does some sanity checks on headers
func (c *Chain) ValidateHeaders(msg *payload.MsgHeaders) error {
	table := database.NewTable(c.db, database.HEADER)
	latestHash, err := c.db.Get(database.LATESTHEADER)
	if err != nil {
		return err
	}

	key := latestHash
	headerBytes, err := table.Get(key)

	latestHeader := &payload.BlockHeader{}
	err = latestHeader.Decode(bytes.NewReader(headerBytes))
	if err != nil {
		return err
	}

	sortedHeaders := util.SortHeadersByHeight(msg.Headers)

	// Do checks on headers
	for _, currentHeader := range sortedHeaders {

		if latestHeader == nil {
			// This should not happen as genesis header is added if new
			// database, however we check nonetheless
			return errors.New("Previous header is nil")
		}

		// Check current hash links with previous
		if !bytes.Equal(currentHeader.PrevBlock, latestHeader.Hash) {
			return errors.New("Last header hash != current header previous hash")
		}

		// Check current Index is one more than the previous Index
		if currentHeader.Height != latestHeader.Height+1 {
			return errors.New("Last header height != current header height")
		}

		// Check current timestamp is later than the previous header's timestamp
		// TODO: This check implies that we use one central time on all nodes in whatever timezone.
		//  But even then, because of communication delays this will lead to problems.
		//  Let's not do this check!!
		//if latestHeader.Timestamp > currentHeader.Timestamp {
		//	return errors.New("Timestamp of Previous Header is later than Timestamp of current Header")
		//}

		// NOTE: These are the only non-contextual checks we can do without the blockchain state
		latestHeader = currentHeader
	}
	return nil
}

// AddHeaders writes the headers to the db in batch (Atomic and Isolated)
func (c *Chain) AddHeaders(msg *payload.MsgHeaders) error {
	return c.db.AddHeaders(msg.Headers)

}

// GetHeaders reads the headers from the db
func (c *Chain) GetHeaders(start []byte, stop []byte) ([]*payload.BlockHeader, error) {
	return c.db.ReadHeaders(start, stop)
}

// GetLatestHeaderHash gets the latest added header of the block chain
func (c *Chain) GetLatestHeaderHash() ([]byte, error) {
	return c.db.Get(database.LATESTHEADER)
}

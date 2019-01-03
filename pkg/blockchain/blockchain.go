package blockchain

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"sort"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var (
	ErrNoBlockchainDb    = errors.New("Blockchain database is not available.")
	ErrInitialisedCheck  = errors.New("Could not check if blockchain db is already initialised.")
	ErrBlockValidation   = errors.New("Block failed sanity check.")
	ErrBlockVerification = errors.New("Block failed to be consistent with the current blockchain.")
)

// Blockchain holds the state of the chain
type Chain struct {
	db  *database.BlockchainDB
	net protocol.Magic
}

func New(net protocol.Magic) (*Chain, error) {
	db, err := database.NewBlockchainDB("test") //TODO: external path configuration
	if err != nil {
		fmt.Println(err)
		return nil, ErrNoBlockchainDb
	}

	marker := []byte("HasBeenInitialisedAlready")
	initialised, err := db.Has(marker)
	if err != nil {
		return nil, ErrInitialisedCheck
	}

	//TODO: Find out if initialisation check can be done with check on existing genesis block
	if !initialised {
		// This is a new db
		fmt.Println("New Blockchain database initialisation")
		db.Put(marker, []byte{})

		// We add the genesis block into the db
		// along with the indexes for it
		if net == protocol.MainNet {

			genesisBlock, err := hex.DecodeString(GenesisBlock)
			if err != nil {
				fmt.Println("Could not add genesis header into db")
				db.Delete(marker)
				return nil, err
			}
			r := bytes.NewReader(genesisBlock)
			b := payload.Block{}
			err = b.Decode(r)

			if err != nil {
				fmt.Println("Could not Decode genesis block")
				db.Delete(marker)
				return nil, err
			}
			err = db.AddHeader(b.Header)
			if err != nil {
				fmt.Println("Could not add genesis header")
				db.Delete(marker)
				return nil, err
			}

			// Cast []Payload to []Stealth
			stealths := make([]transactions.Stealth, len(b.Txs))
			for i, t := range b.Txs {
				stealths[i] = t.(transactions.Stealth)
			}

			err = db.AddTransactions(b.Header.Hash, stealths)
			if err != nil {
				fmt.Println("Could not add Genesis Transactions")
				db.Delete(marker)
				return nil, err
			}
		}
		if net == protocol.TestNet {
			fmt.Println("TODO: Setup the genesisBlock for TestNet")
			return nil, nil
		}
	}

	return &Chain{db, net}, nil
}

func (c *Chain) AddBlock(msg *payload.MsgBlock) error {
	if !validateBlock(msg) {
		return ErrBlockValidation
	}

	if !c.verifyBlock(msg) {
		return ErrBlockVerification
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

//addHeaders is not safe for concurrent access
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

	// Sort the headers
	sortedHeaders := msg.Headers
	sort.Slice(sortedHeaders,
		func(i, j int) bool {
			return sortedHeaders[i].Height < sortedHeaders[j].Height
		})

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

func (c *Chain) AddHeaders(msg *payload.MsgHeaders) error {
	for _, header := range msg.Headers {
		if err := c.db.AddHeader(header); err != nil {
			return err
		}
	}
	return nil
}

func (c *Chain) GetHeaders(start []byte, stop []byte) ([]*payload.BlockHeader, error) {
	return c.db.ReadHeaders(start, stop)
}

func (c *Chain) GetLatestHeaderHash() ([]byte, error) {
	return c.db.Get(database.LATESTHEADER)
}

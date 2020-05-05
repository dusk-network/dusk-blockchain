package candidate

import (
	"testing"
)

func TestGenerateGenesis(t *testing.T) {

	// TODO: KEYS the key generation from the SEED uses rusk atm. Also, the
	// genesis needs to be regenerated accordingly. This test should be
	// adjusted to become an integration test

	/*
		seed := make([]byte, 64)
		if _, err := rand.Read(seed); err != nil {
			t.Fatal(err.Error())
		}

		w := transactions.NewKeyPair(seed)

		// Generate a new genesis block with new wallet pubkey
		genesisHex, err := GenerateGenesisBlock(w.PubKey)
		if err != nil {
			t.Fatalf("expecting valid genesis block")
		}

		// Decode the result hex value to ensure it's a valid block
		blob, err := hex.DecodeString(genesisHex)
		if err != nil {
			t.Fatalf("expecting valid hex %s", err.Error())
		}

		var buf bytes.Buffer
		buf.Write(blob)

		b := block.NewBlock()
		if err := message.UnmarshalBlock(&buf, b); err != nil {
			t.Fatalf("expecting decodable hex %s", err.Error())
		}

		t.Logf("genesis: %s", genesisHex)
	*/
}

func TestGenesisBlock(t *testing.T) {
	// TODO: rework for RUSK integration
	/*
		// read the hard-coded genesis blob for testnet
		blob, err := hex.DecodeString(cfg.TestNetGenesisBlob)
		if err != nil {
			t.Fatalf("expecting valid cfg.TestNetGenesisBlob %s", err.Error())
		}

		// decode the blob to a block
		var buf bytes.Buffer
		buf.Write(blob)

		b := block.NewBlock()
		if err := message.UnmarshalLegacyBlock(&buf, b); err != nil {
			t.Fatalf("expecting decodable cfg.TestNetGenesisBlob %s", err.Error())
		}

		// sanity checks
		if b.Header.Height != 0 {
			t.Fatalf("expecting valid height in cfg.TestNetGenesisBlob")
		}

		if b.Header.Version != 0 {
			t.Fatalf("expecting valid version in cfg.TestNetGenesisBlob")
		}

		if b.Txs[0].Type() != transactions.CoinbaseType {
			t.Fatalf("expecting coinbase tx in cfg.TestNetGenesisBlob")
		}

		// t.Logf
		// t.Logf("GenesisBlock:%s", res)
	*/
}

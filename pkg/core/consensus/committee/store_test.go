package committee

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/stretchr/testify/assert"
)

var table = []struct {
	round        uint64
	step         uint8
	size         int
	expectedSize int
}{
	{1, 1, 3, 1},
	{1, 1, 3, 1},
	{1, 2, 3, 2},
	{1, 3, 3, 3},
	{2, 1, 3, 1},
}

// Test that a committee cache keeps copies of produced voting committees, and
// ensure that the size is as expected.
func TestUpsertCommitteeCache(t *testing.T) {
	e := setupExtractorTest()

	// add some provisioners
	addNewProvisioners(&e.Stakers, 3, 10, 1000)

	// run tests with predefined parameters
	for _, params := range table {
		e.UpsertCommitteeCache(params.round, params.step, params.size)
		assert.Equal(t, params.expectedSize, len(e.committeeCache))
	}
}

func addNewProvisioner(p *user.Provisioners, stake uint64, endHeight uint64, k user.Keys) {
	member := &user.Member{}
	member.PublicKeyEd = k.EdPubKeyBytes
	member.PublicKeyBLS = k.BLSPubKeyBytes
	member.Stakes = make([]user.Stake, 1)
	member.Stakes[0].Amount = stake
	member.Stakes[0].EndHeight = endHeight
	p.Members[string(k.BLSPubKeyBytes)] = member
	p.Set.Insert(k.BLSPubKeyBytes)
	p.TotalWeight += stake
}

func addNewProvisioners(p *user.Provisioners, amount int, stake uint64, endHeight uint64) {
	for i := 0; i < amount; i++ {
		k, _ := user.NewRandKeys()
		addNewProvisioner(p, stake, endHeight, k)
	}
}

func setupExtractorTest() *Extractor {
	provisioners := user.Provisioners{
		Provisioners: *user.NewProvisioners(),
	}
	e := NewExtractor()
	e.Stakers = provisioners

	return e
}

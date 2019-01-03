package generator_test

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	generator "gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof/generators"
)

func TestGeneratorsLen(t *testing.T) {

	point := ristretto.Point{}
	point.SetBase()

	generators := generator.New(point.Bytes())

	generators.Compute(64)

	assert.Equal(t, 64, len(generators.Bases))

}
func TestGeneratorsClear(t *testing.T) {

	gens := generator.New([]byte("some data"))

	gens.Compute(64)
	expected := gens.Bases

	gens.Compute(64)
	actual := gens.Bases

	assert.NotEqual(t, expected, actual)

	gens.Clear()

	gens.Compute(64)
	actual = gens.Bases

	assert.Equal(t, expected, actual)

}

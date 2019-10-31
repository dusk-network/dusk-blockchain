package generation

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
)

// Factory creates a first step reduction Component
type Factory struct {
}

// NewFactory instantiates a Factory
func NewFactory() *Factory {
	return &Factory{}
}

// Instantiate a generation component
func (f *Factory) Instantiate() consensus.Component {
	return NewComponent()
}

package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
)

func (b *broker) CollectAgreementEvent(m bytes.Buffer, hdr *header.Header) error {
	ev := New()
	if err := Unmarshal(&m, ev); err != nil {
		return err
	}

	b.accumulator.Process(ev)
	return nil
}

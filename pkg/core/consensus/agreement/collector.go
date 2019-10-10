package agreement

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
)

func (a *agreement) CollectAgreementEvent(m bytes.Buffer, hdr *header.Header) error {
	ev := New()
	if err := Unmarshal(&m, ev); err != nil {
		return err
	}

	a.accumulator.Process(ev)
	return nil
}

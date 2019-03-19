package reduction

import "gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

type reductionQueue map[uint64]map[uint8][]wire.Event

func newReductionQueue() reductionQueue {
	return reductionQueue{}
}

func (s reductionQueue) GetMessages(round uint64, step uint8) []wire.Event {
	if s[round][step] != nil {
		messages := s[round][step]
		s[round][step] = nil
		return messages
	}

	return nil
}

func (s reductionQueue) PutMessage(round uint64, step uint8, m wire.Event) {
	// Initialise the map on this round if it was not yet created
	if s[round] == nil {
		s[round] = make(map[uint8][]wire.Event)
	}

	s[round][step] = append(s[round][step], m)
}

func (s reductionQueue) Clear(round uint64) {
	s[round] = nil
}

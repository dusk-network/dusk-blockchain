package reduction

type reductionQueue map[uint64]map[uint8][]*reductionEvent

func newReductionQueue() reductionQueue {
	return reductionQueue{}
}

func (s reductionQueue) GetMessages(round uint64, step uint8) []*reductionEvent {
	if s[round][step] != nil {
		messages := s[round][step]
		s[round][step] = nil
		return messages
	}

	return nil
}

func (s reductionQueue) PutMessage(round uint64, step uint8, m *reductionEvent) {
	// Initialise the map on this round if it was not yet created
	if s[round] == nil {
		s[round] = make(map[uint8][]*reductionEvent)
	}

	s[round][step] = append(s[round][step], m)
}

func (s reductionQueue) Clear(round uint64) {
	s[round] = nil
}

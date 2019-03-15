package reduction

type reductionQueue map[uint64]map[uint8][]reductionMessage

func newReductionQueue() reductionQueue {
	return reductionQueue{}
}

func (s reductionQueue) GetMessages(round uint64, step uint8) []reductionMessage {
	if s[round][step] != nil {
		messages := s[round][step]
		s[round][step] = nil
		return messages
	}

	return nil
}

func (s reductionQueue) PutMessage(round uint64, step uint8, m reductionMessage) {
	// Initialise the map on this round if it was not yet created
	if s[round] == nil {
		s[round] = make(map[uint8][]reductionMessage)
	}

	s[round][step] = append(s[round][step], m)
}

func (s reductionQueue) Clear(round uint64) {
	s[round] = nil
}

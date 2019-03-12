package collection

type scoreQueue map[uint64]map[uint8][]*scoreMessage

func newScoreQueue() scoreQueue {
	return scoreQueue{}
}

func (s scoreQueue) GetMessages(round uint64, step uint8) []*scoreMessage {
	if s[round][step] != nil {
		return s[round][step]
	}

	return nil
}

func (s scoreQueue) PutMessage(round uint64, step uint8, m *scoreMessage) {
	// Initialise the map on this round if it was not yet created
	if s[round] == nil {
		s[round] = make(map[uint8][]*scoreMessage)
	}

	s[round][step] = append(s[round][step], m)
}

func (s scoreQueue) Clear(round uint64) {
	s[round] = nil
}

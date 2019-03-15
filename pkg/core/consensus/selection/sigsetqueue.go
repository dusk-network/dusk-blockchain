package selection

type sigSetQueue map[uint64]map[uint8][]*sigSetMessage

func newSigSetQueue() sigSetQueue {
	return sigSetQueue{}
}

func (s sigSetQueue) GetMessages(round uint64, step uint8) []*sigSetMessage {
	if s[round][step] != nil {
		messages := s[round][step]
		s[round][step] = nil
		return messages
	}

	return nil
}

func (s sigSetQueue) PutMessage(round uint64, step uint8, m *sigSetMessage) {
	// Initialise the map on this round if it was not yet created
	if s[round] == nil {
		s[round] = make(map[uint8][]*sigSetMessage)
	}

	s[round][step] = append(s[round][step], m)
}

func (s sigSetQueue) Clear(round uint64) {
	s[round] = nil
}

package topics

/*
import (
	"bytes"
	"fmt"
)

type Message struct {
	Topic   topics.Topic
	Payload interface{}
}

func newMsg(t topics.Topic) *Message {
	return &Message{Topic: t}
}

func Unmarshal(b *bytes.Buffer) (Message, err) {
	var err error 
	topic, err := topics.Extract(b)
	if err != nil {
		return nil, err
	}

	msg := newMsg(topic)
	switch topic {
	case Tx:
		err = UnmarshalTxMessage(b, msg)
	case Candidate:
		err= unmarshalCandidateMessage(b, msg)
	case Score:
		err = unmarshalScoreMessage(b, msg)
	case Reduction:
		err = unmarshalReductionMessage(b, msg)
	case Agreement:
		err = unmarshalAgreementMessage(b, msg)
	case RoundResults:
		err = unmarshalRoundResultMessage(b, msg)
	default:
		err = fmt.Errorf("unsupported topic: %s\n", topic.String())
	}

	if err != nil {
		return nil, err
	}

	return msg, nil
}
*/

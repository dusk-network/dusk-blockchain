package logger

import (
	"bytes"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
)

func (l *LogProcessor) PublishRoundEvent(ab []byte) {
	a := bytes.NewBuffer(ab)
	unmarshaller := agreement.NewUnMarshaller()
	ev, err := unmarshaller.Deserialize(a)
	ae := ev.(*agreement.Agreement)
	if err != nil {
		l.WithTime(log.Fields{
			"code": "round",
		}).WithError(err).Errorln("Cannot unmarshal agreement event")
		return
	}

	l.WithAgreement(ae).Infoln("New Round Published")
}

func (l *LogProcessor) withRoundCode(fields log.Fields) *log.Entry {
	return l.WithTime(fields).WithField("code", "round")
}

func (l *LogProcessor) WithAgreement(ae *agreement.Agreement) *log.Entry {
	fields := log.Fields{
		"round":     ae.Round,
		"step":      ae.Step,
		"blockHash": string(ae.BlockHash),
	}
	entry := l.withRoundCode(fields)

	if l.lastInfo != nil {
		blockTime := time.Since(l.lastInfo.Time)
		entry = entry.WithField("blockTime", blockTime)
		l.lastInfo = &blockInfo{
			Time:      time.Now(),
			Agreement: ae,
		}
	}

	return entry
}

package capi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/tidwall/buntdb"
)

var (
	ProvisionersPrefix = "provisioner"
	RoundInfoPrefix    = "roundinfo"
	EventQueuePrefix   = "eventqueue"
)

func GetKey(name string, value interface{}) string {
	return fmt.Sprintf("%s:%v", name, value)
}

func FetchProvisioners(height uint64) (*user.Provisioners, error) {
	var provisioners user.Provisioners
	err := DBInstance.View(func(t *buntdb.Tx) error {
		key := GetKey(ProvisionersPrefix, height)
		provisionersStr, err := t.Get(key)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		buf.WriteString(provisionersStr)

		provisioners, err = user.UnmarshalProvisioners(buf)
		return err
	})
	return &provisioners, err
}

func StoreProvisioners(provisioners *user.Provisioners, height uint64) error {
	err := DBInstance.Update(func(tx *buntdb.Tx) error {

		var init []byte
		buf := bytes.NewBuffer(init)

		err := user.MarshalProvisioners(buf, provisioners)
		if err != nil {
			return err
		}

		key := GetKey(ProvisionersPrefix, height)

		_, _, err = tx.Set(key, buf.String(), &buntdb.SetOptions{Expires: true, TTL: time.Duration(cfg.Get().API.ExpirationTime) * time.Second})
		return err
	})
	return err
}

func FetchRoundInfo(height uint64) (RoundInfoJSON, error) {

	var targetJSON RoundInfoJSON
	err := DBInstance.View(func(t *buntdb.Tx) error {
		key := GetKey(RoundInfoPrefix, height)
		eventQueueJSONStr, err := t.Get(key)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		buf.WriteString(eventQueueJSONStr)

		err = json.Unmarshal(buf.Bytes(), &targetJSON)
		return err
	})
	return targetJSON, err
}

func StoreRoundInfo(round uint64, step uint8, methodName, name string) error {
	err := DBInstance.Update(func(tx *buntdb.Tx) error {
		eventQueueKey := GetKey(RoundInfoPrefix, fmt.Sprintf("%d:%d", round, step))

		eventQueueJSON := RoundInfoJSON{
			Step:      step,
			UpdatedAt: time.Now(),
			Method:    methodName,
			Name:      name,
		}

		eventQueueByteArr, err := json.Marshal(eventQueueJSON)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(eventQueueKey, string(eventQueueByteArr), &buntdb.SetOptions{Expires: true, TTL: time.Duration(cfg.Get().API.ExpirationTime) * time.Second})
		return err
	})
	return err
}

func FetchEventQueue(height uint64) (EventQueueJSON, error) {
	var eventQueueJSON EventQueueJSON
	err := DBInstance.View(func(t *buntdb.Tx) error {
		key := GetKey(EventQueuePrefix, height)
		eventQueueJSONStr, err := t.Get(key)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		buf.WriteString(eventQueueJSONStr)

		err = json.Unmarshal(buf.Bytes(), &eventQueueJSON)
		return err
	})
	return eventQueueJSON, err
}

func StoreEventQueue(round uint64, step uint8, m message.Message) error {
	err := DBInstance.Update(func(tx *buntdb.Tx) error {
		eventQueueKey := GetKey(EventQueuePrefix, fmt.Sprintf("%d:%d", round, step))

		eventQueueJSON := EventQueueJSON{
			Round:     round,
			Step:      step,
			Message:   m,
			UpdatedAt: time.Now(),
		}

		eventQueueByteArr, err := json.Marshal(eventQueueJSON)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(eventQueueKey, string(eventQueueByteArr), &buntdb.SetOptions{Expires: true, TTL: time.Duration(cfg.Get().API.ExpirationTime) * time.Second})
		return err
	})
	return err
}

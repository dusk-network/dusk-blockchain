package capi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/tidwall/buntdb"
)

var (
	HeaderPrefix       = "header"
	ProvisionersPrefix = "provisioner"
)

func GetKey(name string, value interface{}) string {
	return fmt.Sprintf("%s:%v", name, value)
}

func FetchProvisioners(t *buntdb.Tx, height uint64) (*user.Provisioners, error) {
	key := GetKey(ProvisionersPrefix, height)
	provisionersStr, err := t.Get(key)
	if err != nil || len(provisionersStr) == 0 {
		return nil, err
	}

	buf := new(bytes.Buffer)
	buf.WriteString(provisionersStr)

	provisioners, err := user.UnmarshalProvisioners(buf)

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

func StoreRoundInfo(round uint64, step uint8, m message.Message) error {
	err := DBInstance.Update(func(tx *buntdb.Tx) error {
		eventQueueKey := GetKey("eventqueue", fmt.Sprintf("%d:%d", round, step))

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
func FetchRoundInfo(*buntdb.Tx, uint64) ([]byte, error) {
	return nil, errors.New("method not implemented")
}

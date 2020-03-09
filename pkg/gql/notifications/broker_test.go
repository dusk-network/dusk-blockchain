package notifications

import (
	"container/list"
	"strconv"
	"testing"
)

func TestReapClients(t *testing.T) {

	b := Broker{}
	b.clients = list.New()

	closeCounter := 0
	addedClients := 128

	for i := 0; i < addedClients; i++ {
		c := &wsClient{}
		c.id = strconv.Itoa(i)

		if i%2 == 0 {
			c.closed = 1
			closeCounter++
		}

		b.clients.PushBack(c)
	}

	b.reap()

	for e := b.clients.Front(); e != nil; e = e.Next() {
		c := e.Value.(*wsClient)

		if c.IsClosed() {
			t.Fatalf("client %s was not reaped", c.id)
		}
	}

	if b.clients.Len()+closeCounter != addedClients {
		t.Fatalf("Not all closed")
	}
}

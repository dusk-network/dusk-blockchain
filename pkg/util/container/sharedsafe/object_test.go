package sharedsafe

import (
	"strconv"
	"sync"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
)

func TestObject(t *testing.T) {

	val := 3
	c1 := copyable{v: &val}

	obj := Object{}
	obj.Set(c1)

	c2 := obj.Get().(copyable)

	if *c1.v != *c2.v {
		t.Fatal("getting object returns different value")
	}
}

func TestObjectAsync(t *testing.T) {
	val := 3
	c1 := copyable{v: &val}

	obj := Object{}
	obj.Set(c1)

	// reader
	reader := func(obj *Object, wg *sync.WaitGroup) {

		c := obj.Get().(copyable)

		// data should be always multiples of 3
		if *c.v%3 != 0 {
			panic("not equal " + strconv.Itoa(*c.v))
		}
		// this should modify the copy only
		*c.v = *c.v + 1
		wg.Done()
	}

	// writer modifes the shared data
	writer := func(obj *Object, wg *sync.WaitGroup) {

		c := obj.Get().(copyable)
		*c.v *= 3
		obj.Set(c)

		wg.Done()
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go writer(&obj, &wg)
		go reader(&obj, &wg)
	}

	wg.Wait()
}

type copyable struct {
	v *int
}

func (c copyable) Copy() payload.Safe {
	v := *c.v
	return copyable{v: &v}
}

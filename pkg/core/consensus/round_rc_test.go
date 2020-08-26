// +build race

package consensus

import (
	"sync"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Tests below are available only on race detection enabled
func TestInstantiateStore(t *testing.T) {
	c, _ := initCoordinatorTest(t, topics.Reduction)
	var wg sync.WaitGroup
	wg.Add(2)
	startChan := make(chan bool)
	go func() {
		<-startChan
		for i := 0; i < 100; i++ {
			c.SendInternally(topics.Block, nil, 0)
		}
		wg.Done()
	}()

	go func() {
		<-startChan
		for i := 0; i < 100; i++ {
			c.reinstantiateStore()
		}
		wg.Done()
	}()

	close(startChan)
	wg.Wait()
}

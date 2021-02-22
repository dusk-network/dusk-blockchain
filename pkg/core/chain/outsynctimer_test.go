// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"sync"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

func TestBasicOutSyncTimer(t *testing.T) {
	assert := assert.New(t)

	done := make(chan bool)
	onExpired := func() error {
		done <- true
		return nil
	}

	begin := time.Now().Unix()
	timer := newSyncTimer(2*time.Second, onExpired)
	timer.Start("")
	<-done

	end := time.Now().Unix()

	assert.GreaterOrEqual(end-begin, int64(2))
}

func TestCancelOutSyncTimer(t *testing.T) {
	assert := assert.New(t)

	done := make(chan bool)
	onExpired := func() error {
		done <- true
		return nil
	}

	st := newSyncTimer(500*time.Millisecond, onExpired)
	st.Start("")
	st.Cancel()

	select {
	case <-done:
		assert.Fail("timer not canceled")
	case <-time.After(1 * time.Second):
	}
}

func TestResetOutSyncTimer(t *testing.T) {
	assert := assert.New(t)

	done := make(chan int64)
	onExpired := func() error {
		done <- time.Now().Unix()
		return nil
	}

	st := newSyncTimer(3*time.Second, onExpired)
	st.Start("")

	begin := time.Now().Unix()
	time.Sleep(2 * time.Second)
	st.Reset("")

	end := <-done
	assert.GreaterOrEqual(end-begin, int64(5))
}

// TestConcurrentOutSyncTimer ensures all exposed methods are concurrency-safe.
func TestConcurrentOutSyncTimer(t *testing.T) {
	onExpired := func() error {
		return nil
	}
	timer := newSyncTimer(2*time.Second, onExpired)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		for i := 0; i <= 10; i++ {
			timer.Start("")
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i <= 10; i++ {
			timer.Reset("")
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i <= 10; i++ {
			timer.Cancel()
		}
		wg.Done()
	}()

	wg.Wait()
}

package reduction_test

// var timeOut = 4000 * time.Millisecond
//
// func TestStress(t *testing.T) {
// 	eventBus, rpcBus, _, _, k := launchReductionTest(true, 25)
//
// 	// subscribe for the voteset
// 	voteSetChan := make(chan bytes.Buffer, 1)
// 	l := eventbus.NewChanListener(voteSetChan)
// 	eventBus.Subscribe(topics.ReductionResult, l)
//
// 	// Because round updates are asynchronous (sent through a channel), we wait
// 	// for a bit to let the broker update its round.
// 	time.Sleep(200 * time.Millisecond)
// 	hash, _ := crypto.RandEntropy(32)
//
// 	// Do 10 reduction cycles in a row
// 	for i := 0; i < 10; i++ {
// 		go launchCandidateVerifier(rpcBus, false)
// 		go func() {
// 			// Blast the reducer with many more events than quorum, to see if anything will sneak in
// 			for i := 1; i <= 50; i++ {
// 				go sendReductionBuffers(k, hash, 1, uint8(i), eventBus)
// 			}
// 		}()
// 		sendSelection(1, hash, eventBus)
//
// 		voteSetBuf := <-voteSetChan
// 		// The vote set buffer will have a round as it's first item. Let's read it and discard it
// 		var n uint64
// 		if err := encoding.ReadUint64LE(&voteSetBuf, &n); err != nil {
// 			t.Fatal(err)
// 		}
//
// 		// Unmarshal votesBytes and check them for correctness
// 		unmarshaller := reduction.NewUnMarshaller()
// 		voteSet, err := unmarshaller.UnmarshalVoteSet(&voteSetBuf)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
//
// 		// Make sure we have an equal amount of votes for each step, and no events snuck in where they don't belong
// 		stepMap := make(map[uint8]int)
// 		for _, vote := range voteSet {
// 			stepMap[vote.(*reduction.Reduction).Step%2]++
// 		}
//
// 		assert.Equal(t, stepMap[1], stepMap[0])
// 	}
// }
//
// // Test that the reduction phase works properly in the standard conditions.
// func TestReduction(t *testing.T) {
// 	eventBus, rpcBus, streamer, _, k := launchReductionTest(true, 2)
// 	go launchCandidateVerifier(rpcBus, false)
//
// 	// Because round updates are asynchronous (sent through a channel), we wait
// 	// for a bit to let the broker update its round.
// 	time.Sleep(200 * time.Millisecond)
// 	// send a hash to start reduction
// 	hash, _ := crypto.RandEntropy(32)
// 	sendSelection(1, hash, eventBus)
//
// 	// send mocked events until we get a result from the outgoingAgreement channel
// 	sendReductionBuffers(k, hash, 1, 1, eventBus)
// 	sendReductionBuffers(k, hash, 1, 2, eventBus)
//
// 	timer := time.AfterFunc(1*time.Second, func() {
// 		t.Fatal("")
// 	})
//
// 	for i := 0; i < 2; i++ {
// 		if _, err := streamer.Read(); err != nil {
// 			t.Fatal(err)
// 		}
// 	}
//
// 	timer.Stop()
// }
//
// // Test that the reducer does not send any messages when it is not part of the committee.
// func TestNoPublishingIfNotInCommittee(t *testing.T) {
// 	eventBus, _, streamer, _, _ := launchReductionTest(false, 2)
//
// 	// Because round updates are asynchronous (sent through a channel), we wait
// 	// for a bit to let the broker update its round.
// 	time.Sleep(200 * time.Millisecond)
// 	// send a hash to start reduction
// 	hash, _ := crypto.RandEntropy(32)
// 	sendSelection(1, hash, eventBus)
//
// 	// Try to read from the stream, and see if we get any reduction messages from
// 	// ourselves.
// 	go func() {
// 		for {
// 			_, err := streamer.Read()
// 			assert.NoError(t, err)
// 			// HACK: what's the point?
// 			t.Fatal("")
// 		}
// 	}()
//
// 	timer := time.NewTimer(1 * time.Second)
// 	// if we dont get anything after a second, we can assume nothing was published.
// 	<-timer.C
// }
//
// // Test that timeouts in the reduction phase result in proper behavior.
// func TestReductionTimeout(t *testing.T) {
// 	eb, _, streamer, _, _ := launchReductionTest(true, 2)
//
// 	// send a hash to start reduction
// 	hash, _ := crypto.RandEntropy(32)
//
// 	// Because round updates are asynchronous (sent through a channel), we wait
// 	// for a bit to let the broker update its round.
// 	time.Sleep(200 * time.Millisecond)
// 	sendSelection(1, hash, eb)
//
// 	timer := time.After(1 * time.Second)
// 	<-timer
//
// 	stopChan := make(chan struct{})
// 	time.AfterFunc(1*time.Second, func() {
// 		seenTopics := streamer.SeenTopics()
// 		for _, topic := range seenTopics {
// 			if topic == topics.Agreement {
// 				t.Fatal("")
// 			}
// 		}
//
// 		stopChan <- struct{}{}
// 	})
//
// 	<-stopChan
// }
//
// func TestTimeOutVariance(t *testing.T) {
// 	eb, rpcBus, _, _, _ := launchReductionTest(true, 2)
//
// 	// subscribe to reduction results
// 	resultChan := make(chan bytes.Buffer, 1)
// 	l := eventbus.NewChanListener(resultChan)
// 	eb.Subscribe(topics.ReductionResult, l)
//
// 	// Wait a bit for the round update to go through
// 	time.Sleep(400 * time.Millisecond)
//
// 	// measure the time it takes for reduction to time out
// 	start := time.Now()
// 	// send a hash to start reduction
// 	eb.Publish(topics.BestScore, new(bytes.Buffer))
// 	go launchCandidateVerifier(rpcBus, false)
//
// 	// wait for reduction to finish
// 	<-resultChan
// 	elapsed1 := time.Now().Sub(start)
//
// 	// timer should now have doubled
// 	start = time.Now()
// 	eb.Publish(topics.BestScore, new(bytes.Buffer))
// 	// set up another goroutine for verification
// 	go launchCandidateVerifier(rpcBus, false)
//
// 	// wait for reduction to finish
// 	<-resultChan
// 	elapsed2 := time.Now().Sub(start)
//
// 	// elapsed1 * 2 should be roughly the same as elapsed2
// 	assert.InDelta(t, elapsed1.Seconds()*2, elapsed2.Seconds(), 0.1)
//
// 	// update round
// 	eb.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(2, nil, nil))
//
// 	// Wait a bit for the round update to go through
// 	time.Sleep(200 * time.Millisecond)
//
// 	start = time.Now()
// 	// send a hash to start reduction
// 	eb.Publish(topics.BestScore, new(bytes.Buffer))
// 	// set up another goroutine for verification
// 	go launchCandidateVerifier(rpcBus, false)
//
// 	// wait for reduction to finish
// 	<-resultChan
// 	elapsed3 := time.Now().Sub(start)
//
// 	// elapsed1 and elapsed3 should be roughly the same
// 	assert.InDelta(t, elapsed1.Seconds(), elapsed3.Seconds(), 0.05)
// }
//
// func launchCandidateVerifier(rpcBus *rpcbus.RPCBus, failVerification bool) {
// 	v := make(chan rpcbus.Request, 1)
// 	rpcBus.Register(rpcbus.VerifyCandidateBlock, v)
// 	for {
// 		r := <-v
// 		if failVerification {
// 			r.RespChan <- rpcbus.Response{bytes.Buffer{}, errors.New("verification failed")}
// 		} else {
// 			r.RespChan <- rpcbus.Response{bytes.Buffer{}, nil}
// 		}
// 	}
// }
//
// func launchReductionTest(inCommittee bool, amount int) (*eventbus.EventBus, *rpcbus.RPCBus, *eventbus.GossipStreamer, key.ConsensusKeys, []key.ConsensusKeys) {
// 	eb, streamer := eventbus.CreateGossipStreamer()
// 	k, _ := key.NewRandConsensusKeys()
// 	rpcBus := rpcbus.New()
// 	launchReduction(eb, k, timeOut, rpcBus)
// 	// update round
// 	p, keys := consensus.MockProvisioners(amount)
// 	if inCommittee {
// 		member := consensus.MockMember(k)
// 		p.Set.Insert(k.BLSPubKeyBytes)
// 		p.Members[string(k.BLSPubKeyBytes)] = member
// 	}
//
// 	eb.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(1, p, nil))
//
// 	return eb, rpcBus, streamer, k, keys
// }
//
// // Convenience function, which launches the reduction component and removes the
// // preprocessors for testing purposes (bypassing the republisher and the validator).
// // This ensures proper handling of mocked Reduction events.
// func launchReduction(eb *eventbus.EventBus, k key.ConsensusKeys, timeOut time.Duration, rpcBus *rpcbus.RPCBus) {
// 	reduction.Launch(eb, k, timeOut, rpcBus)
// 	eb.RemoveProcessors(topics.Reduction)
// }
//
// func sendReductionBuffers(keys []key.ConsensusKeys, hash []byte, round uint64, step uint8, eventBus *eventbus.EventBus) {
// 	for i := 0; i < len(keys); i++ {
// 		ev := reduction.MockReductionBuffer(keys[i], hash, round, step)
// 		eventBus.Publish(topics.Reduction, ev)
// 	}
// }
//
// func sendSelection(round uint64, hash []byte, eventBus *eventbus.EventBus) {
// 	bestScoreBuf := selection.MockSelectionEventBuffer(round, hash)
// 	eventBus.Publish(topics.BestScore, bestScoreBuf)
// }

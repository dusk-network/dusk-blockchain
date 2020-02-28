package diagnostics

import (
	"errors"

	_ "net/http/pprof"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

var (
	errAlreadyStarted = errors.New("profile already exits")
)

// ProfileSet allows fetching program samples of different types.
// It could be handful if node user observes problematic situation or any type of inefficiency.
// In most cases, it will allow a developer to catch a perf issue (Memory, CPU or concurrency inefficiency) at development phase.
// As support for periodical sample fetching is added, the tool could enable 'continuous profiling' with negligible overhead.
//
// See also func profile.loop() for a few examples of use
//
type ProfileSet struct {
	profiles map[string]Profile
	quit     chan struct{}
}

// NewProfileSet creates and starts ProfileSet from a set of settings strings
func NewProfileSet() ProfileSet {

	return ProfileSet{
		profiles: make(map[string]Profile),
		quit:     make(chan struct{}),
	}
}

func (s *ProfileSet) drop(name string) {

	item, ok := s.profiles[name]
	if ok {
		close(item.quit)
		delete(s.profiles, name)
	}
}

func (ps *ProfileSet) Spawn(p Profile) error {

	if err := isSupported(p.name); err != nil {
		return err
	}

	if p.n == 0 || p.d == 0 {
		return errors.New("invalid settings")
	}

	_, ok := ps.profiles[p.name]
	if !ok {
		ps.profiles[p.name] = p
		// Start profile lifecycle
		go p.loop()

	} else {
		return errAlreadyStarted
	}

	return nil
}

// Listen listens rpcbus commands to allow enabling/disabling
// any profile in runtime (e.g via rpc)
func (s *ProfileSet) Listen(rpc *rpcbus.RPCBus) {

	/*
		startCmdChan := make(chan rpcbus.Request, 1)
		if err := rpc.Register(rpcbus.StartProfile, startCmdChan); err != nil {
			log.Error(err)
			return
		}

		stopCmdChan := make(chan rpcbus.Request, 1)
		if err := rpc.Register(rpcbus.StopProfile, stopCmdChan); err != nil {
			log.Error(err)
			return
		}

		for {
			select {
			case r := <-startCmdChan:
				settings, err := encoding.ReadString(&r.Params)
				if err != nil {
					r.RespChan <- rpcbus.Response{Resp: bytes.Buffer{}, Err: err}
					continue
				}

				item, err := newProfile(settings)
				if err == nil {
					err = s.spawn(item)
				}

				r.RespChan <- rpcbus.Response{Resp: bytes.Buffer{}, Err: err}

			case r := <-stopCmdChan:

				name, err := encoding.ReadString(&r.Params)
				if err == nil {
					s.drop(name)
				}

				r.RespChan <- rpcbus.Response{Resp: bytes.Buffer{}, Err: err}
			case <-s.quit:

				// Signal all profile loops that it's time to terminate
				for name := range s.profiles {
					s.drop(name)
				}
				return
			}

		}
	*/
}

func (p ProfileSet) Close() {
	close(p.quit)
}

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	_ "net/http/pprof"
)

const (
	blockProfileRate = 1000000000
	//TODO: Find the optimal fraction value for mutex profiling
	mutexProfileRate = 1
)

type profSettings struct {
	n, d  int
	start bool
}

// ProfileSet allows fetching program samples of different types.
// It could be handful if node user observe problematic situation or any type of inefficiency.
// In most cases, it will allow a developer to catch a perf issue (Memory, CPU or concurrency inefficiency) at development phase.
// As support for periodical sample fetching is added, the tool could enable 'continuous profiling' with negligible overhead.
//
// See also func loop() for a few examples of use
//
// TBD: Enable/Disable profiles on demand (via rpc or cli command)
type ProfileSet struct {
	profiles map[string]profSettings
	quit     chan struct{}
}

func NewProfileSet(profiles map[string]profSettings) ProfileSet {

	p := ProfileSet{
		quit: make(chan struct{}),
	}

	p.profiles = profiles
	for name, item := range p.profiles {
		go loop(name, item.n, item.d, item.start, p.quit)
	}

	return p
}

func (p ProfileSet) Close() {
	// Signal all profile loops that it's time to terminate
	close(p.quit)
}

// loop runs a loop for periodical (CPU, Heap etc .. ) samples fetching
// Each N seconds loop restarts a profiler and keep samples fetching for D seconds.
//
// Example use:
//
// name: "cpu", n: 1800, d: 30, s: 0
// Continuous profiling sutiable for profiling a node
// in production with less perf overhead:
// This restarts CPU profiling each half an hour and keep samples fetching for 30 seconds
// output: cpu_$timestamp.prof file
//
// name: "memstats", n: 1800, d: 1, s: 0
// Memory leaks detection with less perf overhead
// This restarts Memstats fetching each half an hour and records GC/Memory stats into logs
//
// name: "heap", n: 1800, d: 1800, s: 1
// Detailed memory inefficiency detection with highest perf overhead
// This records complete memory profile each 30 mins and stores it in mem_$timestamp.pprof file
// suitable in development
// output: heap_$timestamp.prof file
//
// name: "cpu", n: 1800, d: 1800, s: 1
// Detailed CPU inefficiency detection with highest perf overhead
// This records complete CPU profile each 30 mins.
// suitable in development
// output: cpu_$timestamp.prof file
//
// name: "block", n: 1800, d: 30, s: 1
// Measures time in which goroutines is on idle. This should help to find any bottlenecks or
// deadlocks.
// (suitable in development)
// output: block_$timestamp.prof file
//
// name: "mutex", n: 1800, d: 30, s: 1
// This starts contended mutex profile from node startup.
// Mutex profile allows you to capture a fraction of the stack traces of
// goroutines with contended mutexes. (find too wide protected regions)
// (suitable in development)
// output: mutex_$timestamp.prof file

func loop(name string, n, d int, s bool, quit chan struct{}) {

	var err error
	t := time.NewTicker(time.Duration(n) * time.Second)

	// Trigger sampling at startup
	var f *os.File
	if s {
		f, err = startProfiling(name)
		if err != nil {
			// not supported type
			return
		}
	}
	defer stopProfiling(f, name)

	// Restart the sampling each #interval minutes
	for {
		select {
		case <-t.C:

			/// Close previous sampling and start a new one
			stopProfiling(f, name)
			f, err = startProfiling(name)
			if err != nil {
				return
			}

			// Sampling lasts not more than Duration seconds
			t2 := time.NewTicker(time.Duration(d) * time.Second)

			select {
			case <-t2.C:
				stopProfiling(f, name)
				f = nil
			case <-quit:
				return
			}

		case <-quit:
			return
		}
	}
}

// startProfiling initializes the profile selected by name and starts samples
// fetching
func startProfiling(name string) (*os.File, error) {

	createFile := func(name string) *os.File {
		pprofFile, err := os.Create(profFile(name))
		if err != nil {
			log.Errorf("Could not create file for profiling %s", name)
			return nil
		}

		log.WithFields(log.Fields{
			"process": "profile",
			"file":    pprofFile.Name(),
		}).Infof("%s profile starting", name)

		return pprofFile
	}

	// Perform different initializing methods according to the type of profile
	switch name {
	case "mutex":
		runtime.SetMutexProfileFraction(mutexProfileRate)
		return createFile(name), nil
	case "block":
		runtime.SetBlockProfileRate(blockProfileRate)
		return createFile(name), nil
	case "heap", "goroutine":
		return createFile(name), nil
	case "memstats":
		// No file needed for custom sampling that records into logger
		logMemstatsSample()
	case "cpu":
		f := createFile(name)
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Error("Could not start CPU profile: ", err)
		}
		return f, nil
	default:
		err := fmt.Errorf("Unsupported profile name %s", name)
		log.Error(err)
		return nil, err
	}

	return nil, nil
}

// stopProfiling stores profile sampling and resets any profile-related state
func stopProfiling(f *os.File, name string) {

	saveFile := func(f *os.File, name string) {
		p := pprof.Lookup(name)
		if p != nil && f != nil {
			if err := p.WriteTo(f, 0); err != nil {
				log.Errorf("Error on writing profile name %s: %v", name, err)
			} else {
				log.WithFields(log.Fields{
					"process": "profile",
					"file":    f.Name(),
				}).Infof("%s profile saved", name)
			}
		}
	}

	defer func() {
		if f != nil {
			f.Close()
			f = nil
		}
	}()

	// Perform different storing methods according to the type of profile
	switch name {
	case "heap":
		runtime.GC() // get up-to-date statistics
		saveFile(f, name)
	case "mutex":
		saveFile(f, name)
		runtime.SetMutexProfileFraction(0)
	case "block":
		saveFile(f, name)
		runtime.SetBlockProfileRate(0)
	case "goroutine":
		saveFile(f, name)
	case "memstats":
	case "cpu":
		pprof.StopCPUProfile()
		log.WithFields(log.Fields{
			"process": "profile",
			"file":    f.Name,
		}).Infof("% profile saved", name)
	default:
		log.Errorf("Unsupported profile name %s", name)
		return
	}
}

func profFile(prefix string) string {
	name := prefix
	name += "_"
	name += strconv.Itoa(int(time.Now().Unix()))
	return name + ".prof"
}

// Custom profile samples

// logMemstatsSample records memory/GC statistics as log entries
func logMemstatsSample() {

	l := log.WithField("process", "memstats")

	runtime.GC() // get up-to-date statistics

	memStats := new(runtime.MemStats)
	runtime.ReadMemStats(memStats)

	var gcStats debug.GCStats
	debug.ReadGCStats(&gcStats)

	s := memStats
	l.Infof("# runtime.MemStats")
	l.Infof("# Alloc = %d\n", s.Alloc)
	l.Infof("# TotalAlloc = %d\n", s.TotalAlloc)
	l.Infof("# Sys = %d\n", s.Sys)
	l.Infof("# Lookups = %d\n", s.Lookups)
	l.Infof("# Mallocs = %d\n", s.Mallocs)
	l.Infof("# Frees = %d\n", s.Frees)
	l.Infof("# HeapAlloc = %d\n", s.HeapAlloc)
	l.Infof("# HeapSys = %d\n", s.HeapSys)
	l.Infof("# HeapIdle = %d\n", s.HeapIdle)
	l.Infof("# HeapInuse = %d\n", s.HeapInuse)
	l.Infof("# HeapReleased = %d\n", s.HeapReleased)
	l.Infof("# HeapObjects = %d\n", s.HeapObjects)
	l.Infof("# Stack = %d / %d\n", s.StackInuse, s.StackSys)

	// Record GC pause history, most recent 5 entries
	l.Infof("# Stop-the-world Pause time")
	for i, v := range gcStats.Pause {
		l.Infof("# gcStats.Pause[%d] = %d ns", i, v)
		if i == 5 {
			break
		}
	}

}

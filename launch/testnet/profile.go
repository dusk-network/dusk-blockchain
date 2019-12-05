package main

import (
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	_ "net/http/pprof"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
)

type profileMngr struct {
	cpuFile *os.File
	memFile *os.File
}

func newProfile() (*profileMngr, error) {
	p := new(profileMngr)

	// Enable http profiling server if requested
	if cfg.Get().Prof.Address != "" {
		go func() {
			listenAddr := cfg.Get().Prof.Address
			log.Infof("Creating profiling server listening on %s", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			err := http.ListenAndServe(listenAddr, nil)
			log.Error(err)
		}()
	}

	var err error

	// Write cpu profile if requested.
	if cfg.Get().Prof.CPUFile != "" {
		log.Infoln("starting cpu profiler")
		go func() {
			// Run the CPU profiler every half hour, for a minute
			ticker := time.NewTicker(30 * time.Minute)
			for i := 0; ; i++ {
				<-ticker.C
				// Write each profile to a different file
				p.cpuFile, err = os.Create(cfg.Get().Prof.CPUFile + strconv.Itoa(i))
				if err != nil {
					log.WithField("process", "cpu profiling").WithError(err).Errorln("could not create file for profiling")
					return
				}

				log.WithFields(log.Fields{
					"process": "cpu profiling",
					"file":    p.cpuFile.Name(),
				}).Infoln("writing profile")
				if err := pprof.StartCPUProfile(p.cpuFile); err != nil {
					p.cpuFile.Close()
					log.WithField("process", "cpu profiling").WithError(err).Errorln("could not start cpu profiling")
					return
				}

				time.Sleep(1 * time.Minute)
				pprof.StopCPUProfile()
				p.cpuFile.Close()
				p.cpuFile = nil
			}
		}()
	}

	// Write mem profile if requested.
	if cfg.Get().Prof.MemFile != "" {
		log.Infoln("starting memory profiler")
		go func() {
			// Run the memory profiler every half hour
			ticker := time.NewTicker(30 * time.Minute)
			for i := 0; ; i++ {
				<-ticker.C
				p.memFile, err = os.Create(cfg.Get().Prof.MemFile + strconv.Itoa(i))
				if err != nil {
					log.WithField("process", "memory profiling").WithError(err).Errorln("could not create file for profiling")
					return
				}

				log.WithFields(log.Fields{
					"process": "memory profiling",
					"file":    p.memFile.Name(),
				}).Infoln("writing profile")
				_ = pprof.WriteHeapProfile(p.memFile)
				p.memFile.Close()
				p.memFile = nil
			}
		}()
	}

	return p, nil
}

func (p *profileMngr) close() {
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		p.cpuFile.Close()
	}

	if p.memFile != nil {
		_ = pprof.WriteHeapProfile(p.memFile)
		p.memFile.Close()
	}
}

package main

import (
	"net/http"
	"os"
	"runtime/pprof"

	log "github.com/sirupsen/logrus"

	_ "net/http/pprof"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
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
		p.cpuFile, err = os.Create(cfg.Get().Prof.CPUFile)
		if err != nil {
			return nil, err
		}

		if err := pprof.StartCPUProfile(p.cpuFile); err != nil {
			p.cpuFile.Close()
			return nil, err
		}

		log.Infof("CPU profiling will be written to '%s'", cfg.Get().Prof.CPUFile)
	}

	// Write mem profile if requested.
	if cfg.Get().Prof.MemFile != "" {
		p.memFile, err = os.Create(cfg.Get().Prof.MemFile)
		if err != nil {
			return nil, err
		}

		log.Infof("Memory profiling will be written to '%s'", cfg.Get().Prof.MemFile)
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

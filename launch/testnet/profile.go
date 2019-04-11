package main

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"runtime/pprof"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	_ "net/http/pprof"
)

type profileMngr struct {
	cpuFile *os.File
	memFile *os.File
}

func newProfile() (*profileMngr, error) {

	p := new(profileMngr)

	// Enable http profiling server if requested
	if cfg.Get().Profile.Address != "" {
		go func() {
			listenAddr := cfg.Get().Profile.Address
			log.Infof("Creating profiling server "+
				"listening on %s", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			err := http.ListenAndServe(listenAddr, nil)
			log.Error(err)
		}()
	}

	var err error

	// Write cpu profile if requested.
	if cfg.Get().Profile.CPUFile != "" {
		p.cpuFile, err = os.Create(cfg.Get().Profile.CPUFile)
		if err != nil {
			return nil, err
		}
		err := pprof.StartCPUProfile(p.cpuFile)

		if err != nil {
			p.close()
			return nil, err
		}
	}

	// Write mem profile if requested.
	if cfg.Get().Profile.MemFile != "" {
		p.memFile, err = os.Create(cfg.Get().Profile.MemFile)
		if err != nil {
			return nil, err
		}
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

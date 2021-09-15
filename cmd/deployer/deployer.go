// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	duskPath = os.Getenv("DUSK_BLOCKCHAIN_PATH")
	ruskPath = os.Getenv("RUSK_PATH")
)

const (
	pingTimeout   = 10 * time.Second
	checkInterval = 5 * time.Second
	maxRetryLimit = 5

	unixSocket = "unix"
)

// Deployer is an application that simplifies the procedure of (re)starting a
// dusk-blockchain node (both dusk-blockchain and rusk services). It should also
// facilitate automatic diagnostic of runtime issues.
type Deployer struct {
	ConfigPath string

	services  []Service
	interrupt chan os.Signal

	Context context.Context
	cancel  context.CancelFunc
}

// Run should start all services and facilitate automatic faults diagnostic.
func (d *Deployer) Run() {
	d.interrupt = make(chan os.Signal, 1)
	signal.Notify(d.interrupt, os.Interrupt, syscall.SIGTERM)

	if err := d.startAll(); err != nil {
		log.WithError(err).Fatal("failed to start")
		return
	}

	d.Context, d.cancel = context.WithCancel(context.Background())
	defer d.cancel()

	var next bool

	for {
		s := &d.services[0]
		if next {
			s = &d.services[1]
		}

		if !d.checkService(s) {
			log.Info("node terminated")
			break
		}

		next = !next
	}
}

func (d *Deployer) cleanup(conf *config.Registry) {
	// Clean up procedure
	if conf.RPC.Network == unixSocket {
		path := conf.RPC.Address
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			if err := os.Remove(path); err != nil {
				log.WithError(err).WithField("path", path).
					Warn("couldn't delete unix socket")
			}
		}
	}

	if conf.RPC.Rusk.Network == unixSocket {
		path := conf.RPC.Rusk.Address
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			if err := os.Remove(path); err != nil {
				log.WithError(err).WithField("path", path).
					Warn("couldn't delete unix socket")
			}
		}
	}
}

// validate validates both config and the environment.
func (d *Deployer) validate() bool {
	return true
}

func (d *Deployer) startAll() error {
	log.Info("run a node")

	// Load and validate configuration file
	if _, err := os.Stat(duskPath); os.IsNotExist(err) {
		return fmt.Errorf("DUSK_BLOCKCHAIN_PATH %s does not exist: %w", duskPath, err)
	}

	if _, err := os.Stat(ruskPath); os.IsNotExist(err) {
		return fmt.Errorf("RUSK_PATH %s does not exist: %w", ruskPath, err)
	}

	viper.SetConfigFile(d.ConfigPath)

	if err := viper.ReadInConfig(); err != nil {
		log.WithError(err).
			WithField("path", d.ConfigPath).
			Fatal("couldn't read config file")
	}

	var conf config.Registry
	if err := viper.Unmarshal(&conf); err != nil {
		_, _ = fmt.Fprintln(os.Stdout, "Could not decode config file "+err.Error())
		os.Exit(1)
	}

	ruskAddr := conf.RPC.Rusk.Address
	if conf.RPC.Rusk.Network == unixSocket {
		ruskAddr = "unix://" + conf.RPC.Rusk.Address
	}

	d.validate()

	// Clean up should remove any drawbacks for a normal system (re)start
	// E.g remove closed unix socket files, release busy sockets etc..
	d.cleanup(&conf)

	var err error

	ps := make([]*os.Process, 2)

	// Start dusk service
	ps[0], err = startProcess(duskPath, "--config", d.ConfigPath)
	if err != nil {
		log.WithError(err).
			WithField("process", duskPath).
			Fatal("couldn't start process")
		return nil
	}

	// Start rusk service
	// Optionally, a flag here will switch between mockrusk and native rusk service.
	ps[1], err = startProcess(ruskPath, "mockrusk",
		"--rusknetwork", conf.RPC.Rusk.Network,
		"--ruskaddress", conf.RPC.Rusk.Address,
		"--walletstore", conf.Wallet.Store, "--walletfile", conf.Wallet.File,
		"--configfile", d.ConfigPath)

	if err != nil {
		log.WithError(err).
			WithField("process", ruskPath).
			Fatal("couldn't start process")
		return err
	}

	d.services = make([]Service, 2)

	d.services[0] = Service{
		Name:     "dusk",
		Process:  ps[0],
		Addr:     conf.Gql.Address,
		PingFunc: PingDusk,
	}

	d.services[1] = Service{
		Name:     "rusk",
		Process:  ps[1],
		Addr:     ruskAddr,
		PingFunc: PingRusk,
	}

	return nil
}

// stopAll sends system (interrupt) signal to all processes.
func (d *Deployer) stopAll(s os.Signal) {
	for _, serv := range d.services {
		log := log.WithField("signal", s.String()).
			WithField("process", serv.Process.Pid).
			WithField("service", serv.Name)

		log.Info("send signal")

		if err := serv.Process.Signal(s); err != nil {
			log.WithError(err).Error("failed to run process")
		}

		time.Sleep(2 * time.Second)
	}
}

func (d *Deployer) checkService(s *Service) bool {
	timer := time.NewTimer(checkInterval)
	select {
	case s := <-d.interrupt:
		d.stopAll(s)
		return false
	case <-timer.C:
	}

	return d.controlRun(s)
}

func (d *Deployer) controlRun(s *Service) bool {
	ctx, cancel := context.WithTimeout(d.Context, pingTimeout)

	go func(ctx context.Context, cancel context.CancelFunc, s *Service) {
		defer cancel()

		if err := d.control(ctx, s); err != nil {
			log.WithError(err).
				WithField("pid", s.Process.Pid).
				WithField("name", s.Name).
				WithField("retries", s.retry).
				Warn("check service failed")
		} else {
			log.WithField("pid", s.Process.Pid).
				WithField("name", s.Name).Info("check service done")
		}
	}(ctx, cancel, s)

	select {
	case s := <-d.interrupt:
		d.stopAll(s)
		return false
	case <-ctx.Done():
	}

	return true
}

func (d *Deployer) control(ctx context.Context, s *Service) error {
	var err error
	if err = s.PingFunc(ctx, s.Addr); err != nil {
		s.retry++
		if s.retry == maxRetryLimit {
			s.retry = 0
			// We reach max retry limit.
			// Collect core-dump and try to restart all.
			// Optionally, logger level could be increased as well.
			d.stopAll(syscall.SIGABRT)
			return d.startAll()
		}
	}

	return err
}

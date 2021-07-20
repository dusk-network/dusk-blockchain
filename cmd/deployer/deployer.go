package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Deployer struct {
	processes  []*os.Process
	ConfigPath string

	signal chan os.Signal

	rootCtx context.Context
	cancel  context.CancelFunc
}

func (d *Deployer) cleanup(conf *config.Registry) {
	// Clean up procedure
	if conf.RPC.Network == "unix" {
		os.Remove(conf.RPC.Address)
	}

	if conf.RPC.Rusk.Network == "unix" {
		os.Remove(conf.RPC.Rusk.Address)
	}
}

// validate validates both config and the environment.
func (d *Deployer) validate() bool {
	return true
}

func (d *Deployer) start() error {
	logrus.Info("start dusk-blockchain")

	duskPath := os.Getenv("DUSK_BLOCKCHAIN_PATH")
	ruskPath := os.Getenv("RUSK_PATH")
	// TODO: exists?

	viper.SetConfigFile(d.ConfigPath)

	if err := viper.ReadInConfig(); err != nil {
		logrus.WithError(err).
			WithField("path", d.ConfigPath).
			Fatal("couldn't read config file")
	}

	var conf config.Registry
	if err := viper.Unmarshal(&conf); err != nil {
		_, _ = fmt.Fprintln(os.Stdout, "Could not decode config file "+err.Error())
		os.Exit(1)
	}

	d.validate()

	// Clean up should remove any drawbacks for a noraml system restart
	// E.g remove closed unix socket files
	d.cleanup(&conf)

	var err error

	ps := make([]*os.Process, 2)
	path := viper.ConfigFileUsed()

	// Reset/Verify
	ps[0], err = startProcess(duskPath, "--config", path)
	if err != nil {
		logrus.WithError(err).
			WithField("process", duskPath).
			Fatal("couldn't start process")
		return nil
	}

	ps[1], err = startProcess(ruskPath, "mockrusk",
		"--rusknetwork", conf.RPC.Rusk.Network,
		"--ruskaddress", conf.RPC.Rusk.Address,
		"--walletstore", conf.Wallet.Store, "--walletfile", conf.Wallet.File,
		"--configfile", path)

	if err != nil {
		logrus.WithError(err).
			WithField("process", ruskPath).
			Fatal("couldn't start process")
		return err
	}

	d.processes = ps
	return nil
}

// stop send termination signals to all processes.
func (d Deployer) stop(s os.Signal) {
	for _, p := range d.processes {
		logrus.WithField("signal", s.String()).
			WithField("process", p.Pid).
			Info("send signal")

		if err := p.Signal(s); err != nil {
			logrus.WithField("signal", s.String()).
				WithField("process", p.Pid).
				WithError(err).
				Error("failed to run process")
		}

		time.Sleep(2 * time.Second)
	}
}

// Run runs all services and facilitate automatic faults diagnostic.
func (d Deployer) Run() {
	if err := d.start(); err != nil {
		return
	}

	d.rootCtx, d.cancel = context.WithCancel(context.Background())
	defer d.cancel()

	for {
		if !d.loop() {
			break
		}
	}
}

func (d Deployer) loop() bool {
	ctx, cancel := context.WithTimeout(d.rootCtx, time.Duration(10)*time.Second)
	defer cancel()

	d.control(ctx, 0, "TODO url", nil)

	timer := time.NewTimer(5 * time.Second)

	select {
	case s := <-d.signal:
		d.stop(s)
		return false
	case <-timer.C:
	}

	return true
}

// Signal handles interrupt signal.
func (d Deployer) Signal(s os.Signal) {
	if d.cancel != nil {
		d.cancel()
	}

	d.signal <- s
}

func (d Deployer) control(ctx context.Context, ind int, url string, softLimits []int) {
	// Get memory consumption
	// TODO:

	// heartbeat service
	// FD descriptors
	// Goroutines num
	// Memory
	// HDD

	var memory, softlimit, hardlimit int
	if memory > softlimit {

		// Alarm + log entry

		// Enable Trace Logging
		// TODO:

		// Enable profiling
		// TODO:
		return
	}

	if memory > hardlimit {
		// Alarm + log entry
		// Abort system to collect core dump
		// stopSystem(ps, syscall.SIGABRT)

		// Restart delay
		// Restart delay

		// startNode()
		// TODO:
	}

	// CPU
	// Memory
	// Lock

}

func startProcess(path string, arg ...string) (*os.Process, error) {
	cmd := exec.Command(path, arg...)
	cmd.Env = os.Environ()

	// Redirect both STDOUT and STDERR to this service log file
	cmd.Stdout = logrus.StandardLogger().Writer()
	cmd.Stderr = logrus.StandardLogger().Writer()

	logrus.WithField("path", path).
		Info("start process")

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd.Process, nil
}

package engine

import (
	"fmt"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"time"
)

type Network struct {
	Nodes     []*DuskNode
	processes []*os.Process
}

// Bootstrap performs all actions needed to initialize and start a local network
// This network is alive by the end of all tests execution
func (n *Network) Bootstrap(workspace string) error {

	initProfiles()

	blockchainExec, blindBidExec, seederExec, err := n.getExec()
	if err != nil {
		return err
	}

	// Start voucher seeder
	if len(seederExec) > 0 {
		if err := n.start("", seederExec); err != nil {
			return err
		}
	} else {
		// If path not provided, then it's assumed that the seeder is already running
		log.Warnf("Seeder path not provided.")
	}

	// Foreach node read localNet.Nodes, configure and run new nodes
	for i, node := range n.Nodes {

		// create node folder
		nodeDir := workspace + "/node-" + node.Id
		err := os.Mkdir(nodeDir, os.ModeDir|os.ModePerm)
		if err != nil {
			return err
		}

		node.Dir = nodeDir

		// Load wallet path as walletX.dat are hard-coded for now
		// Later they could be generated on the fly per each test execution
		walletsPath, _ := os.Getwd()
		walletsPath += "/../data/"

		// Generate node default config file
		tomlFilePath, err := n.generateConfig(i, walletsPath)
		if err != nil {
			return err
		}

		// Run dusk-blockchain node process
		if err := n.start(nodeDir, blockchainExec, "--config", tomlFilePath); err != nil {
			return err
		}

		// Run blindbid node process
		if err := n.start(nodeDir, blindBidExec); err != nil {
			return err
		}
	}

	log.Infof("Local network workspace: %s", workspace)
	log.Infof("Running %d nodes", len(n.Nodes))

	time.Sleep(10 * time.Second)

	return nil
}

func (n *Network) Teardown() {
	for _, process := range n.processes {
		_ = process.Kill()
	}
}

// generateConfig loads config profile assigned to this node
// It's based on viper global var so it cannot be called concurrently
func (n *Network) generateConfig(nodeIndex int, walletPath string) (string, error) {

	node := n.Nodes[nodeIndex]

	// Load config profile
	profileFunc, ok := profileList[node.ConfigProfileID]
	if !ok {
		return "", fmt.Errorf("invalid config profile for node index %d", nodeIndex)
	}

	profileFunc(nodeIndex, node, walletPath)

	configPath := node.Dir + "/dusk.toml"
	if err := viper.WriteConfigAs(configPath); err != nil {
		return "", fmt.Errorf("config profile err '%s' for node index %d", err.Error(), nodeIndex)
	}

	// Load back
	var err error
	node.Cfg, err = config.LoadFromFile(configPath)
	if err != nil {
		return "", fmt.Errorf("LoadFromFile %s failed with err %s", configPath, err.Error())
	}

	return configPath, nil
}

// Start an OS process with TMPDIR=nodeDir, manageable by the network
func (n *Network) start(nodeDir string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Env = append(cmd.Env, "TMPDIR="+nodeDir)
	if err := cmd.Start(); err != nil {
		return err
	} else {
		n.processes = append(n.processes, cmd.Process)
	}

	return nil
}

// getExec returns paths of all node executables
// dusk-blockchain, blindbid and seeder
func (n *Network) getExec() (string, string, string, error) {

	blockchainExec, err := getEnv("DUSK_BLOCKCHAIN")
	if err != nil {
		return "", "", "", err
	}

	blindBidExec, err := getEnv("DUSK_BLINDBID")
	if err != nil {
		return "", "", "", err
	}

	seederExec, _ := getEnv("DUSK_SEEDER")

	return blockchainExec, blindBidExec, seederExec, nil
}

func getEnv(envVarName string) (string, error) {

	execPath := os.Getenv(envVarName)
	if len(execPath) == 0 {
		return "", fmt.Errorf("ENV variable %s is not declared", envVarName)
	}

	if _, err := os.Stat(execPath); os.IsNotExist(err) {
		return "", fmt.Errorf("ENV variable %s points at non-existing file", envVarName)
	}

	return execPath, nil
}

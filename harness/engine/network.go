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
func (n *Network) Bootstrap(workspace string) {

	initProfiles()

	blockchainExec, blindBidExec, seederExec := n.getExec()

	// Start voucher seeder
	if len(seederExec) > 0 {
		n.start("", seederExec)
	}

	// Foreach node read localNet.Nodes, configure and run new nodes
	for i, node := range n.Nodes {

		// create node folder
		nodeDir := workspace + "/node-" + node.Id
		err := os.Mkdir(nodeDir, os.ModeDir|os.ModePerm)
		if err != nil {
			fmt.Printf(err.Error())
			break
		}

		node.Dir = nodeDir

		// Load wallet path as walletX.dat are hard-coded for now
		// Later they could be generated on the fly per each test execution
		walletsPath, _ := os.Getwd()
		walletsPath += "/../data/"

		// Generate node default config file
		tomlFilePath := n.generateConfig(i, walletsPath)

		// Run dusk-blockchain node process
		n.start(nodeDir, blockchainExec, "--config", tomlFilePath)

		// Run blindbid node process
		n.start(nodeDir, blindBidExec)
	}

	log.Infof("Local network workspace: %s", workspace)
	log.Infof("Running %d nodes", len(n.Nodes))

	// TODO: WaitFor consensus reaching block 2
	time.Sleep(10 * time.Second)
}

func (n *Network) Teardown() {
	for _, process := range n.processes {
		process.Kill()
	}
}

func (n *Network) generateConfig(index int, walletPath string) string {

	node := n.Nodes[index]

	profileFunc, ok := profileList[node.ConfigProfileID]
	if !ok {
		panic("invalid config profile")
	}

	profileFunc(index, node, walletPath)

	configPath := node.Dir + "/dusk.toml"
	viper.WriteConfigAs(configPath)

	// Load back
	var err error
	node.Cfg, err = config.LoadFromFile(configPath)
	if err != nil {
		fmt.Printf("LoadFromFile %s failed with err %s", configPath, err.Error())
	}

	return configPath
}

// Start an OS process with TMPDIR=nodeDir, manageable by the network
func (n *Network) start(nodeDir string, name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Env = append(cmd.Env, "TMPDIR="+nodeDir)
	if err := cmd.Start(); err != nil {
		fmt.Printf(err.Error())
	} else {
		n.processes = append(n.processes, cmd.Process)
	}
}

// getExec returns paths of all node executables
// dusk-blockchain, blindbid, voucher
func (n *Network) getExec() (string, string, string) {
	return os.Getenv("DUSK_BLOCKCHAIN"), os.Getenv("DUSK_BLINDBID"), os.Getenv("DUSK_SEEDER")
}

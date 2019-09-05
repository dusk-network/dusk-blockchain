package cli

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"strconv"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
)

func showHelp(args []string) {
	if args != nil && len(args) > 0 {
		helpStr, ok := commandInfo[args[0]]
		if !ok {
			fmt.Fprintf(os.Stdout, "%v is not a supported command\n", args[0])
			return
		}

		fmt.Fprintln(os.Stdout, helpStr)
		return
	}

	commands := make([]string, 0)
	for cmd, desc := range commandInfo {
		commands = append(commands, cmd+": "+desc)
	}

	sort.Strings(commands)
	for _, cmd := range commands {
		fmt.Fprintln(os.Stdout, "\t"+cmd+"\n")
	}
}

func stopNode() {
	fmt.Fprintln(os.Stdout, "stopping node")

	// Send an interrupt signal to the running process
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		// This should never ever happen
		panic(err)
	}

	if err := p.Signal(os.Interrupt); err != nil {
		// Neither should this
		panic(err)
	}
}

func intToScalar(amount int64) ristretto.Scalar {
	var x ristretto.Scalar
	x.SetBigInt(big.NewInt(amount))
	return x
}

func stringToScalar(s string) (ristretto.Scalar, error) {
	sFloat, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return ristretto.Scalar{}, err
	}

	sInt := int64(math.Floor(sFloat * float64(config.DUSK)))
	return intToScalar(sInt), nil
}

func stringToInt64(s string) (int64, error) {
	sInt, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return (int64(sInt)), nil
}

func stringToUint64(s string) (uint64, error) {
	sInt, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return (uint64(sInt)), nil
}

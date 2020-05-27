package main

import (
	"fmt"
	"github.com/dusk-network/dusk-blockchain/harness/engine"
	"github.com/machinebox/graphql"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	lastBlockUpdate    time.Time
	currentBlock       *Block
	localNet           engine.Network
	duskInfo           *DuskInfo
	node               *engine.DuskNode
	currentBlockNumber uint64
	gqlClient          *graphql.Client
	pendingTx          int
	//rpcClient       *Client
)

type DuskInfo struct {
	ContractsCreated   int64
	TokenTransfers     int64
	ContractCalls      int64
	DuskTransfers      int64
	BlockSize          float64
	LoadTime           float64
	TotalDusk          *big.Int
	EffectiveBlockTime int64
}

func init() {
	duskInfo = new(DuskInfo)
	duskInfo.TotalDusk = big.NewInt(0)

}

func main() {
	defer handlePanic()

	node = engine.NewDuskNode(9503, 9000, "default")
	localNet.Nodes = append(localNet.Nodes, node)

	// Instantiate graphQL client
	gqlClient = graphql.NewClient("http://" + node.Cfg.Gql.Address + "/graphql")

	//// Instantiate gRPC client
	//rpcClient = InitRPCClients(context.Background(), node.Cfg.RPC.Address)

	go Routine()

	http.HandleFunc("/metrics", MetricsHttp)
	err := http.ListenAndServe("0.0.0.0:9099", nil)
	if err != nil {
		panic(err)
	}

}

func handlePanic() {
	if r := recover(); r != nil {
		_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("%+v", r), "Application Exporter panic")
	}
	time.Sleep(time.Second * 1)
}

func Routine() {

	for {

		t1 := time.Now()

		pendingTx, _ = PendingTransactionCount(gqlClient, nil)
		newBlock, err := GetBlockByNumber(gqlClient, map[string]interface{}{"height": currentBlockNumber + 1})

		if err != nil {
			fmt.Errorf("error: %+v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		currentBlockNumber = newBlock.Header.Height

		if currentBlock == nil {

			lastBlockUpdate = time.Now()
			currentBlock = newBlock
			diff := lastBlockUpdate.Sub(t1)
			fmt.Printf("Received first block #%v\n", currentBlock.Header.Height)

			previousBlockNum := currentBlock.Header.Height - 1

			lastBlock, err := GetBlockByNumber(gqlClient, map[string]interface{}{"height": previousBlockNum})
			if err != nil {
				fmt.Printf("Received Error on block  #%v, Error: %+v", currentBlock.Header.Height, err)
				continue
			}

			newTimestamp, _ := strconv.Atoi(newBlock.Header.Timestamp)
			lastTimestamp, _ := strconv.Atoi(lastBlock.Header.Timestamp)

			duskInfo.EffectiveBlockTime = int64(time.Unix(int64(newTimestamp), 0).Sub(time.Unix(int64(lastTimestamp), 0)).Seconds())
			duskInfo.LoadTime = diff.Seconds()

			continue
		}
		if newBlock.Header.Height > currentBlock.Header.Height {
			fmt.Printf("Received a new block #%v\n", newBlock.Header.Height)
			currentBlock = newBlock
			lastBlockUpdate = time.Now()

			previousBlockNum := currentBlock.Header.Height - 1

			lastBlock, _ := GetBlockByNumber(gqlClient, map[string]interface{}{"height": previousBlockNum})

			newTimestamp, _ := strconv.Atoi(newBlock.Header.Timestamp)
			lastTimestamp, _ := strconv.Atoi(lastBlock.Header.Timestamp)

			duskInfo.EffectiveBlockTime = int64(time.Unix(int64(newTimestamp), 0).Sub(time.Unix(int64(lastTimestamp), 0)).Seconds())

			diff := lastBlockUpdate.Sub(t1)
			duskInfo.LoadTime = diff.Seconds()
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// HTTP response handler for /metrics
func MetricsHttp(w http.ResponseWriter, r *http.Request) {
	var allOut []string

	now := time.Now()

	CalculateTotals(currentBlock)

	var duskTps int64
	if duskInfo.EffectiveBlockTime != 0 {
		duskTps = int64(len(currentBlock.Txs)) / duskInfo.EffectiveBlockTime
	}

	allOut = append(allOut, fmt.Sprintf("dusk_block %v", currentBlock.Header.Height))
	allOut = append(allOut, fmt.Sprintf("dusk_seconds_last_block %0.2f", now.Sub(lastBlockUpdate).Seconds()))
	allOut = append(allOut, fmt.Sprintf("dusk_effective_block_time %v", duskInfo.EffectiveBlockTime))
	allOut = append(allOut, fmt.Sprintf("dusk_block_transactions %v", len(currentBlock.Txs)))
	allOut = append(allOut, fmt.Sprintf("dusk_block_tps %d", duskTps))
	allOut = append(allOut, fmt.Sprintf("dusk_block_value %v", duskInfo.TotalDusk))
	allOut = append(allOut, fmt.Sprintf("dusk_block_size_bytes %v", duskInfo.BlockSize))
	allOut = append(allOut, fmt.Sprintf("dusk_pending_transactions %v", pendingTx))
	allOut = append(allOut, fmt.Sprintf("dusk_contracts_created %v", duskInfo.ContractsCreated))
	allOut = append(allOut, fmt.Sprintf("dusk_token_transfers %v", duskInfo.TokenTransfers))
	allOut = append(allOut, fmt.Sprintf("dusk_transfers %v", duskInfo.DuskTransfers))
	allOut = append(allOut, fmt.Sprintf("dusk_load_time %0.4f", duskInfo.LoadTime))

	fmt.Fprintln(w, strings.Join(allOut, "\n"))
}

func CalculateTotals(block *Block) {
	duskInfo.TotalDusk = big.NewInt(0)
	duskInfo.ContractsCreated = 0
	duskInfo.TokenTransfers = 0
	duskInfo.DuskTransfers = 0

	for _, b := range block.Txs {

		//TODO: calculate contracts created
		//duskInfo.ContractsCreated++

		//TODO: implement other types ?
		if len(b.Data) > 0 {
			duskInfo.TokenTransfers++
		}

		//// TODO: calculate transfers
		//duskInfo.DuskTransfers++

		//var totalDusk int64
		//for _, v := range b.StandardTx().Outputs {
		//	totalDusk = totalDusk + int64(v.Value())
		//}

		//duskInfo.TotalDusk.Add(duskInfo.TotalDusk, big.NewInt(totalDusk))
	}

	//size := strings.Split(currentBlock.Size().String(), " ")
	//duskInfo.BlockSize = stringToFloat(size[0]) * 1000
}

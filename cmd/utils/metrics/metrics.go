package metrics

import (
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dusk-network/dusk-blockchain/harness/engine"
	"github.com/machinebox/graphql"
)

const (
	layoutISO = "2006-01-02 15:04:05 -0700 MST"
)

var (
	lastBlockUpdate    time.Time
	currentBlock       *Block
	localNet           engine.Network
	duskInfo           *DuskInfo
	node               *engine.DuskNode
	currentBlockNumber uint64
	pendingTx          int
	//rpcClient       *Client
)

// DuskInfo is the placeholder for exporter metrics
type DuskInfo struct {
	ContractsCreated   int64
	TokenTransfers     int64
	ContractCalls      int64
	DuskTransfers      int64
	BlockSize          float64
	LoadTime           float64
	TotalDusk          *big.Int
	EffectiveBlockTime int64
	GQLEndpoint        string
	GQLClient          *graphql.Client
}

// RunMetrics will run the metrics collection endpoint
func RunMetrics(gqlPort, nodePort, port int, hostname string) {
	duskInfo = new(DuskInfo)
	duskInfo.TotalDusk = big.NewInt(0)

	node = engine.NewDuskNode(gqlPort, nodePort, "default")
	localNet.Nodes = append(localNet.Nodes, node)

	duskInfo.GQLEndpoint = "http://" + node.Cfg.Gql.Address + "/graphql"
	fmt.Printf("Instantiate graphQL client\n")
	duskInfo.GQLClient = graphql.NewClient(duskInfo.GQLEndpoint)

	// Instantiate gRPC client
	// rpcClient = InitRPCClients(context.Background(), node.Cfg.RPC.Address)

	go Routine()

	http.HandleFunc("/metrics", HandlerMetrics)
	err := http.ListenAndServe(hostname+":"+strconv.Itoa(port), nil)
	if err != nil {
		panic(err)
	}

}

// Routine its a infinite loop that collects metrics
func Routine() {

	for {

		t1 := time.Now()
		var err error
		pendingTx, err = pendingTransactionCount(duskInfo)
		if err != nil {
			fmt.Printf("ERROR: pendingTransactionCount: %+v\n", err)
		}

		//newBlock, err := getBlockByNumber(duskInfo, map[string]interface{}{"height": currentBlockNumber + 1})
		newBlock, err := getLatestBlock(duskInfo)
		if err != nil {
			fmt.Printf("ERROR: getBlockByNumber: %+v\n", err)
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

			lastBlock, err := getBlockByNumber(duskInfo, map[string]interface{}{"height": previousBlockNum})
			if err != nil {
				fmt.Printf("Received Error on block  #%v, Error: %+v", currentBlock.Header.Height, err)
				continue
			}

			newTimestamp, _ := time.Parse(layoutISO, newBlock.Header.Timestamp)
			lastTimestamp, _ := time.Parse(layoutISO, lastBlock.Header.Timestamp)

			duskInfo.EffectiveBlockTime = int64(time.Unix(newTimestamp.Unix(), 0).Sub(time.Unix(lastTimestamp.Unix(), 0)).Seconds())
			duskInfo.LoadTime = diff.Seconds()

			continue
		}
		if newBlock.Header.Height > currentBlock.Header.Height {
			fmt.Printf("Received a new block #%v\n", newBlock.Header.Height)
			currentBlock = newBlock
			lastBlockUpdate = time.Now()

			previousBlockNum := currentBlock.Header.Height - 1

			lastBlock, _ := getBlockByNumber(duskInfo, map[string]interface{}{"height": previousBlockNum})

			newTimestamp, _ := time.Parse(layoutISO, newBlock.Header.Timestamp)
			lastTimestamp, _ := time.Parse(layoutISO, lastBlock.Header.Timestamp)

			duskInfo.EffectiveBlockTime = int64(time.Unix(newTimestamp.Unix(), 0).Sub(time.Unix(lastTimestamp.Unix(), 0)).Seconds())

			diff := lastBlockUpdate.Sub(t1)
			duskInfo.LoadTime = diff.Seconds()
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// HandlerMetrics is a HTTP response handler for /metrics
func HandlerMetrics(w http.ResponseWriter, r *http.Request) {
	var allOut []string

	now := time.Now()

	if currentBlock == nil {
		fmt.Printf("Received Error on HandlerMetrics  #%v, Error: %+v\n", 0, http.StatusServiceUnavailable)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	CalculateTotals(currentBlock)

	var duskTps int64
	if duskInfo.EffectiveBlockTime != 0 {
		duskTps = int64(len(currentBlock.Txs)) / duskInfo.EffectiveBlockTime
	}

	allOut = append(allOut, fmt.Sprintf("dusk_block %v", currentBlock.Header.Height))
	allOut = append(allOut, fmt.Sprintf("dusk_seconds_last_block %0.2f", now.Sub(lastBlockUpdate).Seconds()))
	allOut = append(allOut, fmt.Sprintf("dusk_effective_block_time %v", duskInfo.EffectiveBlockTime))
	allOut = append(allOut, fmt.Sprintf("dusk_block_transactions %v", len(currentBlock.Txs)))
	allOut = append(allOut, fmt.Sprintf("dusk_pending_transactions %v", pendingTx))
	allOut = append(allOut, fmt.Sprintf("dusk_block_tps %d", duskTps))
	allOut = append(allOut, fmt.Sprintf("dusk_block_value %v", duskInfo.TotalDusk))
	allOut = append(allOut, fmt.Sprintf("dusk_block_size_bytes %v", duskInfo.BlockSize))
	allOut = append(allOut, fmt.Sprintf("dusk_contracts_created %v", duskInfo.ContractsCreated))
	allOut = append(allOut, fmt.Sprintf("dusk_token_transfers %v", duskInfo.TokenTransfers))
	allOut = append(allOut, fmt.Sprintf("dusk_transfers %v", duskInfo.DuskTransfers))
	allOut = append(allOut, fmt.Sprintf("dusk_load_time %0.4f", duskInfo.LoadTime))

	_, _ = fmt.Fprintln(w, strings.Join(allOut, "\n"))
}

// CalculateTotals will calculate totals for a block
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

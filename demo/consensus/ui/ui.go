package ui

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	ui "github.com/bcicen/termui"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

type blk struct {
	height int
	sha    string
	tx     string
}

type UI struct {
	Nodes       *sync.Map
	NodeStrings []string
	ChainChan   chan string
	Chain       []string
	LogChan     chan string
	Log         []string
	BlockTime   chan int64
	LastTime    int64
	BlockChan   chan *block.Block
}

type Node struct {
	Online    bool
	Port      string
	Phase     string
	Step      uint32
	Round     uint64
	Weight    uint64
	LastBlock string
}

func Run(u *UI) error {
	// CPU and MEM History
	cpuh := make([]int, 0)
	memh := make([]int, 0)
	bt := make([]int, 0)
	labels := make([]string, 0)
	counter := 1

	// Ugly hack to set the sparkline scale to 100%
	cpuh = append(cpuh, 100)
	memh = append(memh, 100)

	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	strs := []string{}

	// Connected nodes widget
	ls := ui.NewList()
	ls.Items = strs
	ls.ItemFgColor = ui.ColorYellow
	ls.BorderLabel = "Connected Nodes"
	ls.Height = 14

	// Blockchain log
	par := ui.NewPar(" ")
	par.Height = 14
	par.BorderLabel = "Blockchain"

	par2 := ui.NewPar(" ")
	par2.Height = 14
	par2.BorderLabel = "Network Log"

	// Sparklines elements - for system status
	spl0 := ui.NewSparkline()
	spl0.Height = 5
	spl0.Title = "CPU"
	spl0.LineColor = ui.ColorGreen

	spl1 := ui.NewSparkline()
	spl1.Height = 5
	spl1.Title = "MEM"
	spl1.LineColor = ui.ColorRed

	spls1 := ui.NewSparklines(spl0, spl1)
	spls1.Height = 14
	//	spls1.Width = 10
	spls1.BorderLabel = "System"

	// Block time bar chart
	bc := ui.NewBarChart()
	bc.Height = 14
	bc.BorderLabel = "Block Time (seconds)"
	bc.BarColor = ui.ColorBlue
	bc.BarWidth = 8

	// Block info section
	blockInfo := ui.NewPar(" ")
	blockInfo.Height = 14
	blockInfo.BorderLabel = "Last Block Info"

	// Node definitions - move somewhere else
	var nodes []*ui.Par

	node1 := ui.NewPar("[Status:](fg-black) [Offline](fg-red)")
	node1.Height = 9
	node1.BorderLabel = "Node 1 Status"
	node1.Bg = ui.ColorGreen
	node1.TextBgColor = ui.ColorGreen
	node1.TextFgColor = ui.ColorBlack
	nodes = append(nodes, node1)

	node2 := ui.NewPar("[Status:](fg-black) [Offline](fg-red)")
	node2.Height = 9
	node2.BorderLabel = "Node 2 Status"
	node2.Bg = ui.ColorYellow
	node2.TextBgColor = ui.ColorYellow
	node2.TextFgColor = ui.ColorBlack
	nodes = append(nodes, node2)

	node3 := ui.NewPar("[Status:](fg-black) [Offline](fg-red)")
	node3.Height = 9
	node3.BorderLabel = "Node 3 Status"
	node3.Bg = ui.ColorBlue
	node3.TextBgColor = ui.ColorBlue
	node3.TextFgColor = ui.ColorBlack
	nodes = append(nodes, node3)

	node4 := ui.NewPar("[Status:](fg-black) [Offline](fg-red)")
	node4.Height = 9
	node4.BorderLabel = "Node 4 Status"
	node4.Bg = ui.ColorMagenta
	node4.TextBgColor = ui.ColorMagenta
	node4.TextFgColor = ui.ColorBlack
	nodes = append(nodes, node4)

	node5 := ui.NewPar("[Status:](fg-black) [Offline](fg-red)")
	node5.Height = 9
	node5.BorderLabel = "Node 5 Status"
	node5.Bg = ui.ColorCyan
	node5.TextBgColor = ui.ColorCyan
	node5.TextFgColor = ui.ColorBlack
	nodes = append(nodes, node5)

	node6 := ui.NewPar("[Status:](fg-black) [Offline](fg-red)")
	node6.Height = 9
	node6.BorderLabel = "Node 6 Status"
	node6.Bg = ui.ColorWhite
	node6.TextBgColor = ui.ColorWhite
	node6.TextFgColor = ui.ColorBlack
	nodes = append(nodes, node6)

	node7 := ui.NewPar("[Status:](fg-black) [Offline](fg-red)")
	node7.Height = 9
	node7.BorderLabel = "Node 7 Status"
	node7.Bg = ui.ColorBlack
	node7.TextBgColor = ui.ColorBlack
	node7.TextFgColor = ui.ColorWhite
	nodes = append(nodes, node7)

	node8 := ui.NewPar("[Status:](fg-black) [Offline](fg-red)")
	node8.Height = 9
	node8.BorderLabel = "Node 8 Status"
	node8.Bg = ui.ColorRed
	node8.TextBgColor = ui.ColorRed
	node8.TextFgColor = ui.ColorWhite
	nodes = append(nodes, node8)

	ui.Merge("timer", ui.NewTimerCh(500*time.Millisecond))

	// build layout
	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(3, 0, ls),
			ui.NewCol(3, 0, spls1),
			ui.NewCol(3, 0, par),
			ui.NewCol(3, 0, blockInfo)),
		ui.NewRow(
			ui.NewCol(3, 0, node1),
			ui.NewCol(3, 0, node2),
			ui.NewCol(3, 0, node3),
			ui.NewCol(3, 0, node4)),
		ui.NewRow(
			ui.NewCol(3, 0, node5),
			ui.NewCol(3, 0, node6),
			ui.NewCol(3, 0, node7),
			ui.NewCol(3, 0, node8)),
		ui.NewRow(
			ui.NewCol(6, 0, par2),
			ui.NewCol(6, 0, bc)))
	// calculate layout
	ui.Body.Align()

	ui.Render(ui.Body)

	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})
	ui.Handle("/timer/500ms", func(e ui.Event) {

		v, _ := mem.VirtualMemory()
		c, _ := cpu.Percent(0, false)
		memh = append(memh, int(v.UsedPercent))

		cpuh = append(cpuh, int(c[0]))

		//spark.Data = memh
		spls1.Lines[0].Data = cpuh
		spls1.Lines[1].Data = memh

		// Build the blockchain log
		u.UpdateChain()
		if len(u.Chain) > 12 {
			u.Chain = u.Chain[len(u.Chain)-12 : len(u.Chain)]
		}

		var blockchain string
		for _, elem := range u.Chain {

			blockchain += elem
		}

		par.Text = blockchain

		// Update connected nodes
		ls.Items = u.NodeStrings

		// Update the nodes
		u.UpdateNodes(nodes)

		// Update block time graph
		t := u.UpdateBlockTime()
		if t > 0 {
			bt = append(bt, int(t))
			if len(bt) > 13 {
				bt = bt[len(bt)-13 : len(bt)]
			}

			bc.Data = bt
			labels = append(labels, "Block "+strconv.Itoa(counter))
			if len(labels) > 13 {
				labels = labels[len(labels)-13 : len(labels)]
			}
			bc.DataLabels = labels
			counter++
		}

		// Update block info
		info, hash := u.UpdateBlockInfo()
		if info != "" {
			blockInfo.Text = info
			blockInfo.BorderLabel = "Block Info " + hash
		}

		ui.Body.Align()
		ui.Render(ui.Body)
	})

	ui.Handle("/timer/1s", func(e ui.Event) {
		// Network log
		u.UpdateLog()
		if len(u.Log) > 12 {
			u.Log = u.Log[len(u.Log)-12 : len(u.Log)]
		}

		var log string
		for _, elem := range u.Log {
			log += elem
		}

		par2.Text = log
	})

	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		ui.Body.Width = ui.TermWidth()
		ui.Body.Align()
		ui.Clear()
		ui.Render(ui.Body)
	})

	ui.Loop()

	return nil
}

func (u *UI) UpdateNodes(nodes []*ui.Par) {
	for i, node := range nodes {
		num := i + 1
		v, ok := u.Nodes.Load(num)
		if !ok {
			os.Exit(1)
		}

		if !v.(*Node).Online {
			node.Text = "[Status:](fg-black) [Offline](fg-red)"
			continue
		}

		if num < 4 || num > 5 {
			node.Text = "[Port:](fg-black) [" + v.(*Node).Port + "](fg-red)\n[Phase:](fg-black) [" + v.(*Node).Phase + "](fg-red)\n[Round:](fg-black) [" + strconv.Itoa(int(v.(*Node).Round)) + "](fg-red)\n[Step:](fg-black) [" + strconv.Itoa(int(v.(*Node).Step)) + "](fg-red)\n[Weight:](fg-black) [" + strconv.Itoa(int(v.(*Node).Weight)) + "](fg-red)\n[Last Block:](fg-black) [" + v.(*Node).LastBlock + "](fg-red)"
		}

		if num == 4 || num == 5 {
			node.Text = "[Port:](fg-black) [" + v.(*Node).Port + "](fg-white)\n[Phase:](fg-black) [" + v.(*Node).Phase + "](fg-white)\n[Round:](fg-black) [" + strconv.Itoa(int(v.(*Node).Round)) + "](fg-white)\n[Step:](fg-black) [" + strconv.Itoa(int(v.(*Node).Step)) + "](fg-white)\n[Weight:](fg-black) [" + strconv.Itoa(int(v.(*Node).Weight)) + "](fg-white)\n[Last Block:](fg-black) [" + v.(*Node).LastBlock + "](fg-white)"
		}

		if num == 7 {
			node.Text = "[Port:](fg-white) [" + v.(*Node).Port + "](fg-red)\n[Phase:](fg-white) [" + v.(*Node).Phase + "](fg-red)\n[Round:](fg-white) [" + strconv.Itoa(int(v.(*Node).Round)) + "](fg-red)\n[Step:](fg-white) [" + strconv.Itoa(int(v.(*Node).Step)) + "](fg-red)\n[Weight:](fg-white) [" + strconv.Itoa(int(v.(*Node).Weight)) + "](fg-red)\n[Last Block:](fg-white) [" + v.(*Node).LastBlock + "](fg-red)"
		}

		if num == 8 {
			node.Text = "[Port:](fg-white) [" + v.(*Node).Port + "](fg-black)\n[Phase:](fg-white) [" + v.(*Node).Phase + "](fg-black)\n[Round:](fg-white) [" + strconv.Itoa(int(v.(*Node).Round)) + "](fg-black)\n[Step:](fg-white) [" + strconv.Itoa(int(v.(*Node).Step)) + "](fg-black)\n[Weight:](fg-white) [" + strconv.Itoa(int(v.(*Node).Weight)) + "](fg-black)\n[Last Block:](fg-white) [" + v.(*Node).LastBlock + "](fg-black)"
		}
	}
}

func (u *UI) UpdateChain() {
loop:
	for len(u.ChainChan) > 0 {
		msg := <-u.ChainChan
		for _, elem := range u.Chain {
			if strings.Contains(elem, msg) {
				continue loop
			}
		}

		u.Chain = append(u.Chain, msg)
	}
}

func (u *UI) UpdateLog() {
	for len(u.LogChan) > 0 {
		msg := <-u.LogChan
		switch msg[0:4] {
		case "2000":
			msg = "[" + msg + "](fg-green)"
		case "2001":
			msg = "[" + msg + "](fg-yellow)"
		case "2002":
			msg = "[" + msg + "](fg-blue)"
		case "2003":
			msg = "[" + msg + "](fg-magenta)"
		case "2004":
			msg = "[" + msg + "](fg-cyan)"
		case "2005":
			msg = "[" + msg + "](fg-white)"
		case "2006":
			msg = "[" + msg + "](fg-black)"
		case "2007":
			msg = "[" + msg + "](fg-red)"
		}
		u.Log = append(u.Log, msg)
	}
}

func (u *UI) UpdateBlockTime() int64 {
	if len(u.BlockTime) > 0 {
		t := <-u.BlockTime
		elapsed := t - u.LastTime
		u.LastTime = t

		return elapsed
	}

	return 0
}

func (u *UI) UpdateBlockInfo() (string, string) {
	var info string
	var hash string
	for len(u.BlockChan) > 0 {
		info = ""
		blk := <-u.BlockChan
		hash = fmt.Sprintf("%s...", hex.EncodeToString(blk.Header.Hash)[0:16])
		t := time.Unix(blk.Header.Timestamp, 0)
		info += fmt.Sprintf("Timestamp: %s\n", t.Format(time.UnixDate))
		info += fmt.Sprintf("Height: %d\n", blk.Header.Height)
		info += fmt.Sprintf("Seed: %s...\n", hex.EncodeToString(blk.Header.Seed)[0:16])
		info += fmt.Sprintf("Merkle root hash: %s...\n", hex.EncodeToString(blk.Header.TxRoot)[0:16])
	}

	return info, hash
}

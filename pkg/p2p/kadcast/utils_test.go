package kadcast

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"testing"
	"time"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
)

func TestConv(t *testing.T) {
	var slce []byte = []byte{0, 0}
	peerNum := binary.BigEndian.Uint16(slce)
	fmt.Printf("%v", peerNum)
}
func testPOW(t *testing.T) {
	a := Peer{
		ip:   [4]byte{192, 169, 1, 1},
		port: 25519,
		id:   [16]byte{22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22},
	}

	println(a.computePeerNonce())
}

func testUDP(t *testing.T) {
	var port uint16 = 25519
	ip := [4]byte{31, 4, 243, 248}
	router := makeRouter(ip, port)
	buffer := ring.NewBuffer(100)

	
	go startUDPListener("udp", buffer, router)
	destIP := [4]byte{167, 71, 11, 218}
	destPeer := makePeer(destIP, port)
	router.tree.addPeer(router.myPeerInfo, destPeer)

	// Add 4 nodes
	peers := [][4]byte{
		[4]byte{1, 1, 1, 1},
		//[4]byte{2, 2, 2, 2},
		//[4]byte{3, 3, 3, 3},
		//[4]byte{4, 4, 4, 4},
		//[4]byte{5, 5, 5, 5},
		//[4]byte{6, 6, 6, 6},
		//[4]byte{7, 7, 7, 7},
		//[4]byte{8, 8, 8, 8},
		//[4]byte{9, 9, 9, 9},
		//[4]byte{10, 10, 10, 10},
		//[4]byte{11, 11, 11, 11},
		//[4]byte{12, 12, 12, 12},
		//[4]byte{13, 13, 13, 13},
		//[4]byte{14, 14, 14, 14},
		//[4]byte{15, 15, 15, 15},
		//[4]byte{16, 16, 16, 16},
		//[4]byte{17, 17, 17, 17},
		//[4]byte{18, 18, 18, 18},
		//[4]byte{19, 19, 19, 19},
		//[4]byte{20, 20, 20, 20},
		//[4]byte{21, 21, 21, 21},
		//[4]byte{22, 22, 22, 22},
		//[4]byte{23, 23, 23, 23},
		//[4]byte{24, 24, 24, 24},
		//[4]byte{25, 25, 25, 25},
	}

	for _, ip := range peers {
		router.tree.addPeer(router.myPeerInfo, makePeer(ip, port))
	}

	// Send PING packet
	router.sendPing(destPeer)
	router.sendFindNodes()
	//router.sendPing(destPeer)

	for {

	}
}

func testProtocol(t *testing.T) {
	// Our node info.
	var port uint16 = 25519
	ip := [4]byte{62, 57, 180, 247}
	router := makeRouter(ip, port)

	// Create buffer.
	queue := ring.NewBuffer(500)

	// Launch PacketProcessor rutine.
	go processPacket(queue, &router)

	// Launch a listener for our node.
	go startUDPListener("udp", queue, router)

	// Create BootstrapNodes Peer structs
	var port1 uint16 = 25519
	boot1 := makePeer(ip, port1)

	var bootstrapNodes []Peer
	bootstrapNodes = append(bootstrapNodes[:], boot1)

	// Start Bootstrapping process.
	err := initBootstrap(&router, bootstrapNodes)
	if err != nil {
		log.Fatal("Error during the Bootstrap Process. Job terminated.")
	}

	// Once the bootstrap succeeded, start the network discovery.
	startNetworkDiscovery(&router)

	for {

	}
}

func TestUDPConn(t *testing.T) {
	//go func () {
		lAddr := getLocalIPAddress()
		pc, err := net.ListenUDP("udp", &lAddr)
		if err != nil {
			log.Println(err)
		}

		//simple read
		buffer := make([]byte, 1024)
		pc.SetReadDeadline(time.Now().Add(5* time.Second))

		_, _, er := pc.ReadFromUDP(buffer)

		if er != nil {
			log.Printf("%v", er)
		}
	//}()
	for {

	}
}

func TestLRUPolicy(t *testing.T) {
	// Create my router 
	var port uint16 = 25519
	ip := [4]byte{62, 57, 180, 247}
	router := makeRouter(ip, port)

	// Create peers.
	peers := [][4]byte{
		[4]byte{1, 1, 1, 0},
		[4]byte{1, 1, 1, 1},
		[4]byte{1, 1, 1, 2},
		[4]byte{1, 1, 1, 3},
		[4]byte{1, 1, 1, 4},
		[4]byte{1, 1, 1, 5},
		[4]byte{1, 1, 1, 6},
		[4]byte{1, 1, 1, 7},
		[4]byte{1, 1, 1, 8},
		[4]byte{1, 1, 1, 9},
		[4]byte{1, 1, 1, 10},
		[4]byte{1, 1, 1, 11},
		[4]byte{1, 1, 1, 12},
		[4]byte{1, 1, 1, 13},
		[4]byte{1, 1, 1, 14},
		[4]byte{1, 1, 1, 15},
		[4]byte{1, 1, 1, 16},
		[4]byte{1, 1, 1, 17},
		[4]byte{1, 1, 1, 18},
		[4]byte{1, 1, 1, 19},
		[4]byte{1, 1, 1, 20},
		[4]byte{1, 1, 1, 21},
		[4]byte{1, 1, 1, 22},
		[4]byte{1, 1, 1, 23},
		[4]byte{1, 1, 1, 24},
		[4]byte{1, 1, 1, 25},
		[4]byte{1, 1, 1, 26},
		[4]byte{1, 1, 1, 27},
		[4]byte{1, 1, 1, 28},
		[4]byte{1, 1, 1, 29},
		[4]byte{1, 1, 1, 30},
		[4]byte{1, 1, 1, 31},
		[4]byte{1, 1, 1, 32},
		[4]byte{1, 1, 1, 33},
		[4]byte{1, 1, 1, 34},
		[4]byte{1, 1, 1, 35},
		[4]byte{1, 1, 1, 36},
		[4]byte{1, 1, 1, 37},
		[4]byte{1, 1, 1, 38},
		[4]byte{1, 1, 1, 39},
		[4]byte{1, 1, 1, 40},
		[4]byte{1, 1, 1, 41},
		[4]byte{1, 1, 1, 42},
		[4]byte{1, 1, 1, 43},
		[4]byte{1, 1, 1, 44},
		[4]byte{1, 1, 1, 45},
		[4]byte{1, 1, 1, 46},
		[4]byte{1, 1, 1, 47},
		[4]byte{1, 1, 1, 48},
		[4]byte{1, 1, 1, 49},
		[4]byte{1, 1, 1, 50},
		[4]byte{1, 1, 1, 51},
		[4]byte{1, 1, 1, 52},
		[4]byte{1, 1, 1, 53},
		[4]byte{1, 1, 1, 54},
		[4]byte{1, 1, 1, 55},
		[4]byte{1, 1, 1, 56},
		[4]byte{1, 1, 1, 57},
		[4]byte{1, 1, 1, 58},
		[4]byte{1, 1, 1, 59},
		[4]byte{1, 1, 1, 60},
		[4]byte{1, 1, 1, 61},
		[4]byte{1, 1, 1, 62},
		[4]byte{1, 1, 1, 63},
		[4]byte{1, 1, 1, 64},
		[4]byte{1, 1, 1, 65},
		[4]byte{1, 1, 1, 66},
		[4]byte{1, 1, 1, 67},
		[4]byte{1, 1, 1, 68},
		[4]byte{1, 1, 1, 69},
		[4]byte{1, 1, 1, 71},
		[4]byte{1, 1, 1, 72},
		[4]byte{1, 1, 1, 73},
		[4]byte{1, 1, 1, 74},
		[4]byte{1, 1, 1, 75},
		[4]byte{1, 1, 1, 76},
		[4]byte{1, 1, 1, 77},
		[4]byte{1, 1, 1, 78},
		[4]byte{1, 1, 1, 79},
		[4]byte{1, 1, 1, 80},
		[4]byte{1, 1, 1, 81},
		[4]byte{1, 1, 1, 82},
		[4]byte{1, 1, 1, 83},
		[4]byte{1, 1, 1, 84},
		[4]byte{1, 1, 1, 85},
		[4]byte{1, 1, 1, 86},
		[4]byte{1, 1, 1, 87},
		[4]byte{1, 1, 1, 88},
		[4]byte{1, 1, 1, 89},
		[4]byte{1, 1, 1, 90},
		[4]byte{1, 1, 1, 91},
		[4]byte{1, 1, 1, 92},
		[4]byte{1, 1, 1, 93},
		[4]byte{1, 1, 1, 94},
		[4]byte{1, 1, 1, 95},
		[4]byte{1, 1, 1, 96},
		[4]byte{1, 1, 1, 97},
		[4]byte{1, 1, 1, 98},
		[4]byte{1, 1, 1, 99},
		[4]byte{1, 1, 1, 100},
		[4]byte{1, 1, 1, 101},
		[4]byte{1, 1, 1, 102},
		[4]byte{1, 1, 1, 103},
		[4]byte{1, 1, 1, 104},
		[4]byte{1, 1, 1, 105},
		[4]byte{1, 1, 1, 106},
		[4]byte{1, 1, 1, 107},
		[4]byte{1, 1, 1, 108},
		[4]byte{1, 1, 1, 109},
		[4]byte{1, 1, 1, 110},
		[4]byte{1, 1, 1, 111},
		[4]byte{1, 1, 1, 112},
		[4]byte{1, 1, 1, 113},
		[4]byte{1, 1, 1, 114},
		[4]byte{1, 1, 1, 115},
		[4]byte{1, 1, 1, 116},
		[4]byte{1, 1, 1, 117},
		[4]byte{1, 1, 1, 118},
		[4]byte{1, 1, 1, 119},
		[4]byte{1, 1, 1, 120},
		[4]byte{1, 1, 1, 121},
		[4]byte{1, 1, 1, 122},
		[4]byte{1, 1, 1, 123},
		[4]byte{1, 1, 1, 124},
		[4]byte{1, 1, 1, 125},
		[4]byte{1, 1, 1, 126},
		[4]byte{1, 1, 1, 127},
		[4]byte{1, 1, 1, 128},
		[4]byte{1, 1, 1, 129},
		[4]byte{1, 1, 1, 130},
		[4]byte{1, 1, 1, 131},
		[4]byte{1, 1, 1, 132},
		[4]byte{1, 1, 1, 133},
		[4]byte{1, 1, 1, 134},
		[4]byte{1, 1, 1, 135},
		[4]byte{1, 1, 1, 136},
		[4]byte{1, 1, 1, 137},
		[4]byte{1, 1, 1, 138},
		[4]byte{1, 1, 1, 139},
		[4]byte{1, 1, 1, 140},
		[4]byte{1, 1, 1, 141},
		[4]byte{1, 1, 1, 142},
		[4]byte{1, 1, 1, 143},
		[4]byte{1, 1, 1, 144},
		[4]byte{1, 1, 1, 145},
		[4]byte{1, 1, 1, 146},
		[4]byte{1, 1, 1, 147},
		[4]byte{1, 1, 1, 148},
		[4]byte{1, 1, 1, 149},
		[4]byte{1, 1, 1, 150},
		[4]byte{1, 1, 1, 151},
		[4]byte{1, 1, 1, 152},
		[4]byte{1, 1, 1, 153},
		[4]byte{1, 1, 1, 154},
		[4]byte{1, 1, 1, 155},
		[4]byte{1, 1, 1, 156},
		[4]byte{1, 1, 1, 157},
		[4]byte{1, 1, 1, 158},
		[4]byte{1, 1, 1, 159},
		[4]byte{1, 1, 1, 160},
		[4]byte{1, 1, 1, 161},
		[4]byte{1, 1, 1, 162},
		[4]byte{1, 1, 1, 163},
		[4]byte{1, 1, 1, 164},
		[4]byte{1, 1, 1, 165},
		[4]byte{1, 1, 1, 166},
		[4]byte{1, 1, 1, 167},
		[4]byte{1, 1, 1, 168},
		[4]byte{1, 1, 1, 169},
		[4]byte{1, 1, 1, 170},
		[4]byte{1, 1, 1, 171},
		[4]byte{1, 1, 1, 172},
		[4]byte{1, 1, 1, 173},
		[4]byte{1, 1, 1, 174},
		[4]byte{1, 1, 1, 175},
		[4]byte{1, 1, 1, 176},
		[4]byte{1, 1, 1, 177},
		[4]byte{1, 1, 1, 178},
		[4]byte{1, 1, 1, 179},
		[4]byte{1, 1, 1, 180},
		[4]byte{1, 1, 1, 181},
		[4]byte{1, 1, 1, 182},
		[4]byte{1, 1, 1, 183},
		[4]byte{1, 1, 1, 184},
		[4]byte{1, 1, 1, 185},
		[4]byte{1, 1, 1, 186},
		[4]byte{1, 1, 1, 187},
		[4]byte{1, 1, 1, 188},
		[4]byte{1, 1, 1, 189},
		[4]byte{1, 1, 1, 190},
		[4]byte{1, 1, 1, 191},
		[4]byte{1, 1, 1, 192},
		[4]byte{1, 1, 1, 193},
		[4]byte{1, 1, 1, 194},
		[4]byte{1, 1, 1, 195},
		[4]byte{1, 1, 1, 196},
		[4]byte{1, 1, 1, 197},
		[4]byte{1, 1, 1, 198},
		[4]byte{1, 1, 1, 199},
		[4]byte{1, 1, 1, 200},
		[4]byte{1, 1, 1, 201},
		[4]byte{1, 1, 1, 202},
		[4]byte{1, 1, 1, 203},
		[4]byte{1, 1, 1, 204},
		[4]byte{1, 1, 1, 205},
		[4]byte{1, 1, 1, 206},
		[4]byte{1, 1, 1, 207},
		[4]byte{1, 1, 1, 208},
		[4]byte{1, 1, 1, 209},
		[4]byte{1, 1, 1, 210},
		[4]byte{1, 1, 1, 211},
		[4]byte{1, 1, 1, 212},
		[4]byte{1, 1, 1, 213},
		[4]byte{1, 1, 1, 214},
		[4]byte{1, 1, 1, 215},
		[4]byte{1, 1, 1, 216},
		[4]byte{1, 1, 1, 217},
		[4]byte{1, 1, 1, 218},
		[4]byte{1, 1, 1, 219},
		[4]byte{1, 1, 1, 220},
		[4]byte{1, 1, 1, 221},
		[4]byte{1, 1, 1, 222},
		[4]byte{1, 1, 1, 223},
		[4]byte{1, 1, 1, 224},
		[4]byte{1, 1, 1, 225},
		[4]byte{1, 1, 1, 226},
		[4]byte{1, 1, 1, 227},
		[4]byte{1, 1, 1, 228},
		[4]byte{1, 1, 1, 229},
		[4]byte{1, 1, 1, 230},
		[4]byte{1, 1, 1, 231},
		[4]byte{1, 1, 1, 232},
		[4]byte{1, 1, 1, 233},
		[4]byte{1, 1, 1, 234},
		[4]byte{1, 1, 1, 235},
		[4]byte{1, 1, 1, 236},
		[4]byte{1, 1, 1, 237},
		[4]byte{1, 1, 1, 238},
		[4]byte{1, 1, 1, 239},
		[4]byte{1, 1, 1, 240},
		[4]byte{1, 1, 1, 241},
		[4]byte{1, 1, 1, 242},
		[4]byte{1, 1, 1, 243},
		[4]byte{1, 1, 1, 244},
		[4]byte{1, 1, 1, 245},
		[4]byte{1, 1, 1, 246},
		[4]byte{1, 1, 1, 247},
		[4]byte{1, 1, 1, 248},
		[4]byte{1, 1, 1, 249},
		[4]byte{1, 1, 1, 250},
		[4]byte{1, 1, 1, 251},
		[4]byte{1, 1, 1, 252},
		[4]byte{1, 1, 1, 253},
		[4]byte{1, 1, 1, 254},
		[4]byte{1, 1, 1, 255},
	}

	// Add peers to the tree.
	for _, ip := range peers {
		router.tree.addPeer(router.myPeerInfo, makePeer(ip, port))
	}

	fmt.Printf("%v", router.tree.buckets[60].entries)


}

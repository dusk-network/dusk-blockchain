package utils

import (
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/dusk-network/dusk-blockchain/pkg/voucher/client"
	"github.com/dusk-network/dusk-blockchain/pkg/voucher/client/node"
	"github.com/dusk-network/dusk-blockchain/pkg/voucher/client/request"
)

// ChallengeToProof will receive a challenge, a seeder key and return the proof
func ChallengeToProof(challenge *[]byte, seederKey *[]byte) []byte {
	h := sha512.New()
	h.Write(*challenge)
	h.Write(*seederKey)
	return h.Sum(nil)
}

// ReaderToVarLenBytes will attempt to fetch a variable length bytes from a reader
func ReaderToVarLenBytes(reader io.Reader) ([]byte, error) {
	buf := make([]byte, 2)
	_, err := io.ReadFull(reader, buf[:])
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch response payload from the voucher: %s", err)
	}

	size := binary.LittleEndian.Uint16(buf)
	buf = make([]byte, size)
	_, err = io.ReadFull(reader, buf[:])
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch the variable bytes from the voucher: %s", err)
	}

	return buf, nil
}

// ReaderToTCPAddr will attempt to fetch a net.TCPAddr from a reader
func ReaderToTCPAddr(reader io.Reader) (*net.TCPAddr, error) {
	buf := make([]byte, 2)
	_, err := io.ReadFull(reader, buf[:])
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch response payload from the voucher: %s", err)
	}

	size := binary.LittleEndian.Uint16(buf)
	buf = make([]byte, size)
	_, err = io.ReadFull(reader, buf[:])
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch the IP address from the voucher: %s", err)
	}
	ip := net.IP(buf)

	buf = make([]byte, 2)
	_, err = io.ReadFull(reader, buf[:])
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch the Port from the voucher: %s", err)
	}
	port := binary.LittleEndian.Uint16(buf)

	return &net.TCPAddr{IP: ip, Port: int(port), Zone: ""}, nil
}

// PingVoucher will perform a simple command in the server, and return error if any problem
func PingVoucher(voucherAddr *string) error {
	c, err := client.NewClient(*voucherAddr)
	if err != nil {
		return err
	}
	defer c.Close()

	ping, err := request.NewPing()
	if err != nil {
		return err
	}

	err = c.Execute(ping)
	if err != nil {
		return err
	}

	return nil
}

// RegisterNodeToVoucher will attempt to submit the node information to a voucher, and return the net.TCPAddr of the voucher that have accepted the node
func RegisterNodeToVoucher(seederKey *[]byte, voucherAddr *string, nodePubKey *[]byte, nodePort *uint16) (*net.TCPAddr, error) {
	c, err := client.NewClient(*voucherAddr)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	r, err := request.NewRegisterNode(nodePubKey, nodePort)
	if err != nil {
		return nil, err
	}

	err = c.Execute(r)
	if err != nil {
		return nil, err
	}

	parentVoucherTCPAddr, err := ReaderToTCPAddr(c)
	if err != nil {
		return nil, err
	}

	challenge, err := ReaderToVarLenBytes(c)
	if err != nil {
		return nil, err
	}

	proof := ChallengeToProof(&challenge, seederKey)
	cv, err := client.NewClient(parentVoucherTCPAddr.String())
	if err != nil {
		return nil, err
	}
	defer cv.Close()

	rv, err := request.NewRegisterNodeVerifyChallenge(nodePubKey, nodePort, &proof)
	if err != nil {
		return nil, err
	}

	err = cv.Execute(rv)
	if err != nil {
		return nil, err
	}

	return parentVoucherTCPAddr, nil
}

// RequestSeeders will query a voucher for the available seeders
func RequestSeeders(seederKey *[]byte, voucherAddr *string, nodePubKey *[]byte, nodeSocket *net.TCPAddr) (*[]*node.Node, error) {
	c, err := client.NewClient(*voucherAddr)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	voucherTCPAddr, err := net.ResolveTCPAddr("tcp", *voucherAddr)
	if err != nil {
		return nil, err
	}

	nodeSocketFmt := fmt.Sprintf("%v:%v", nodeSocket.IP, nodeSocket.Port)
	voucherTCPAddrFmt := fmt.Sprintf("%v:%v", voucherTCPAddr.IP, voucherTCPAddr.Port)

	h := sha512.New()
	h.Write(*nodePubKey)
	h.Write([]byte(nodeSocketFmt))
	h.Write([]byte(voucherTCPAddrFmt))
	hs := h.Sum(nil)
	proof := ChallengeToProof(&hs, seederKey)

	nodePort := uint16(nodeSocket.Port)
	request, err := request.NewSeeders(nodePubKey, &nodePort, &proof)
	if err != nil {
		return nil, err
	}

	err = c.Execute(request)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 2)
	_, err = io.ReadFull(c, buf[:])
	if err != nil {
		return nil, err
	}

	nodeList := []*node.Node{}
	size := int(binary.LittleEndian.Uint16(buf))
	var pubKey []byte
	var addr *net.TCPAddr
	for i := 0; i < size; i++ {
		// Skipping leading size, not necessary in this context
		buf = make([]byte, 2)
		_, err := io.ReadFull(c, buf[:])
		if err != nil {
			return nil, err
		}

		buf = make([]byte, 2)
		_, err = io.ReadFull(c, buf[:])
		if err != nil {
			return nil, err
		}
		size = int(binary.LittleEndian.Uint16(buf))
		pubKey = make([]byte, size)
		_, err = io.ReadFull(c, pubKey[:])
		if err != nil {
			return nil, err
		}
		addr, err = ReaderToTCPAddr(c)
		if err != nil {
			return nil, err
		}

		nodeList = append(nodeList, node.NewNode(*addr, pubKey))
	}

	return &nodeList, nil
}

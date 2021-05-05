// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package encoding

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"

	"golang.org/x/crypto/blake2b"
)

// PeerBytesSize represents the amount of bytes
// necessary to represent .
const PeerBytesSize int = 22

// PeerInfo stores peer addr and ID.
// A slice of PeerInfo is wired on NODES message.
type PeerInfo struct {
	IP   [4]byte
	Port uint16

	ID [16]byte
}

// MakePeerFromAddr is same as MakePeer but resolve addr with ResolveTCPAddr.
func MakePeerFromAddr(addr string) (PeerInfo, error) {
	laddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return PeerInfo{}, err
	}

	// TODO: IPv6 not supported here yet
	l := len(laddr.IP)

	var ip [4]byte

	if l > 4 {
		copy(ip[:], laddr.IP[l-4 : l][:])
	}

	return MakePeer(ip, uint16(laddr.Port)), nil
}

// MakePeerFromIP from ipaddr (resolvable by ResolveTCPAddr) and port.
func MakePeerFromIP(ipaddr string, port uint16) (PeerInfo, error) {
	laddr, err := net.ResolveTCPAddr("tcp4", ipaddr)
	if err != nil {
		return PeerInfo{}, err
	}

	// TODO: IPv6 not supported here
	l := len(laddr.IP)

	var ip [4]byte

	if l > 4 {
		copy(ip[:], laddr.IP[l-4 : l][:])
	}

	return MakePeer(ip, port), nil
}

// MakePeer builds a peer tuple by computing ID over IP and port.
func MakePeer(ip [4]byte, port uint16) PeerInfo {
	id := computePeerID(ip, port)
	return PeerInfo{ip, port, id}
}

// MarshalBinary marshal peer tuple into binary buffer.
func (peer *PeerInfo) MarshalBinary(buf *bytes.Buffer) error {
	// Marshaling IP as 32bits (IPv4 supported)
	// TODO: Extend IP marshaling to support both IPv4 and IPv6
	if _, err := buf.Write(peer.IP[:]); err != nil {
		return err
	}

	portBytes := make([]byte, 2)
	byteOrder.PutUint16(portBytes, peer.Port)

	if _, err := buf.Write(portBytes); err != nil {
		return err
	}

	if _, err := buf.Write(peer.ID[:]); err != nil {
		return err
	}

	return nil
}

// UnmarshalBinary build peer tuple from binary buffer.
func (peer *PeerInfo) UnmarshalBinary(buf *bytes.Buffer) error {
	if _, err := buf.Read(peer.IP[:]); err != nil {
		return err
	}

	var portBytes [2]byte
	if _, err := buf.Read(portBytes[:]); err != nil {
		return err
	}

	peer.Port = byteOrder.Uint16(portBytes[:])

	// TODO: Do we really need to marshal/unmarshal id
	// Instead, ComputePeerID might be suitable
	if _, err := buf.Read(peer.ID[:]); err != nil {
		return err
	}

	return nil
}

// GetUDPAddr make net.UDPAddr from PeerInfo IP:port.
func (peer PeerInfo) GetUDPAddr() net.UDPAddr {
	if len(peer.IP) < 4 {
		return net.UDPAddr{}
	}

	return net.UDPAddr{
		IP:   net.IPv4(peer.IP[0], peer.IP[1], peer.IP[2], peer.IP[3]),
		Port: int(peer.Port),
		Zone: "",
	}
}

// IsEqual returns true if two peers tuples are identical.
func (peer PeerInfo) IsEqual(p PeerInfo) bool {
	if !bytes.Equal(peer.ID[:], p.ID[:]) {
		return false
	}

	if !bytes.Equal(peer.IP[:], p.IP[:]) {
		return false
	}

	if peer.Port != p.Port {
		return false
	}

	return true
}

// String returns peerinfo address and ID as a string.
func (peer PeerInfo) String() string {
	id := hex.EncodeToString(peer.ID[:])
	if len(id) >= 7 {
		id = id[0:7]
	}

	return fmt.Sprintf("%s, %s", peer.Address(), id)
}

// Address returns peer address as a string.
func (peer PeerInfo) Address() string {
	addr := peer.GetUDPAddr()
	return addr.String()
}

// computePeerID Performs the hash of the wallet public
// IP address and gets the first 16 bytes of it.
func computePeerID(ip [4]byte, port uint16) [16]byte {
	seed := make([]byte, 2)
	binary.LittleEndian.PutUint16(seed, port)

	seed = append(seed, ip[:]...)
	doubleLenID := blake2b.Sum256(seed[:])

	var halfLenID [16]byte
	copy(halfLenID[:], doubleLenID[0:16])

	return halfLenID
}

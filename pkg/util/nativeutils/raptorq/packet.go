// +build

package raptorq

import (
	"bytes"
	"encoding/binary"

	"errors"
)

const (
	MaxUDPLength = 1472
	// SymbolSize is the max length of encoding symbol data unit that can fit into default IP datagram
	SourceObjectIDSize = 8
	SymbolSize         = MaxUDPLength - SourceObjectIDSize - 1 - 2 - 8 - 4
	MaxSubBlockSize    = 3 * SymbolSize
)

type SourceObjectID [SourceObjectIDSize]byte

var (
	byteOrder      = binary.LittleEndian
	ErrTooLargeUDP = errors.New("packet cannot fit into default MTU of 1500")
)

// Packet is the UDP packet that consistes of encoding symbol data unit and raptorq-specific data
type Packet struct {
	sourceObjectID SourceObjectID

	// source block number
	sbn uint8
	// encoding symbol identifier
	esi uint16

	commonOTI     uint64
	schemeSpecOTI uint32

	// payload fields
	symbolData [SymbolSize]byte
}

func NewPacket(sourceObjectID []byte, sbn uint8, esi uint16, symbolData []byte, commonOTI uint64, schemeSpecOTI uint32) Packet {

	p := Packet{
		sbn:           sbn,
		esi:           esi,
		commonOTI:     commonOTI,
		schemeSpecOTI: schemeSpecOTI,
	}

	copy(p.symbolData[:], symbolData)
	copy(p.sourceObjectID[:], sourceObjectID)

	return p
}

func (p *Packet) MarshalBinary(buf *bytes.Buffer) error {

	// source object id
	if _, err := buf.Write(p.sourceObjectID[:]); err != nil {
		return err
	}

	// Source block number
	if err := buf.WriteByte(p.sbn); err != nil {
		return err
	}

	esi := make([]byte, 2)
	byteOrder.PutUint16(esi, p.esi)

	// Write encoding symbol identifier
	if _, err := buf.Write(esi); err != nil {
		return err
	}

	// Write commonOTI
	commonOTI := make([]byte, 8)
	byteOrder.PutUint64(commonOTI, p.commonOTI)

	if _, err := buf.Write(commonOTI); err != nil {
		return err
	}

	// Write schemeSpecOTI
	schemeSpecOTI := make([]byte, 4)
	byteOrder.PutUint32(schemeSpecOTI, p.schemeSpecOTI)

	if _, err := buf.Write(schemeSpecOTI); err != nil {
		return err
	}

	// Write packet data
	if _, err := buf.Write(p.symbolData[:]); err != nil {
		return err
	}

	if buf.Len() > MaxUDPLength {
		return ErrTooLargeUDP
	}

	return nil
}

func (p *Packet) UnmarshalBinary(buf *bytes.Buffer) error {

	// source object id
	sourceObjectID := make([]byte, SourceObjectIDSize)
	if _, err := buf.Read(sourceObjectID); err != nil {
		return err
	}

	sbn, err := buf.ReadByte()
	if err != nil {
		return err
	}

	esi := make([]byte, 2)
	if _, err := buf.Read(esi); err != nil {
		return err
	}

	commonOTI := make([]byte, 8)
	if _, err := buf.Read(commonOTI); err != nil {
		return err
	}

	schemeSpecOTI := make([]byte, 4)
	if _, err := buf.Read(schemeSpecOTI); err != nil {
		return err
	}

	data := make([]byte, buf.Len())
	if _, err := buf.Read(data); err != nil {
		return err
	}

	p.sbn = uint8(sbn)
	p.esi = byteOrder.Uint16(esi)

	p.commonOTI = byteOrder.Uint64(commonOTI)
	p.schemeSpecOTI = byteOrder.Uint32(schemeSpecOTI)
	copy(p.symbolData[:], data)
	copy(p.sourceObjectID[:], sourceObjectID)

	return nil
}

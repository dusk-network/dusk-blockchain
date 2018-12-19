package payload

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// CertData holds a public address and a proof.
type CertData struct {
	PubAddress []byte // 32 bytes
	Proof      []byte // 32 bytes
}

// Step defines the certificate information for a step in the consensus.
type Step struct {
	StepNum uint32
	Data    []*CertData
}

// Certificate defines a block certificate made as a result from the consensus.
type Certificate struct {
	Signature []byte // Batched BLS signatures (32 bytes)
	Steps     []*Step
	Hash      []byte // Certificate hash (32 bytes)
}

// NewCertificate returns a Certificate struct with the provided signature.
func NewCertificate(sig []byte) *Certificate {
	return &Certificate{
		Signature: sig,
	}
}

// AddStep adds a step to the Certificate struct.
func (c *Certificate) AddStep(step *Step) {
	c.Steps = append(c.Steps, step)
}

// NewStep returns a Step struct with StepNum n.
func NewStep(n uint32) *Step {
	return &Step{
		StepNum: n,
	}
}

// AddData will add a combination of a public address and a proof as a CertData
// struct into Step s.
func (s *Step) AddData(pubAddr, proof []byte) {
	data := &CertData{
		PubAddress: pubAddr,
		Proof:      proof,
	}

	s.Data = append(s.Data, data)
}

// SetHash will set the Certificate hash.
func (c *Certificate) SetHash() error {
	buf := new(bytes.Buffer)
	if err := c.EncodeHashable(buf); err != nil {
		return err
	}

	h, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return err
	}

	c.Hash = h
	return nil
}

// EncodeHashable will encode all fields needed from the CertificateStruct to create
// a certificate hash. Result will be written to w.
func (c *Certificate) EncodeHashable(w io.Writer) error {
	if err := encoding.Write256(w, c.Signature); err != nil {
		return err
	}

	lSteps := uint64(len(c.Steps))
	if err := encoding.WriteVarInt(w, lSteps); err != nil {
		return err
	}

	for _, step := range c.Steps {
		if err := encoding.WriteUint32(w, binary.LittleEndian, step.StepNum); err != nil {
			return err
		}

		lData := uint64(len(step.Data))
		if err := encoding.WriteVarInt(w, lData); err != nil {
			return err
		}

		for _, data := range step.Data {
			if err := encoding.Write256(w, data.PubAddress); err != nil {
				return err
			}

			if err := encoding.Write256(w, data.Proof); err != nil {
				return err
			}
		}
	}

	return nil
}

// Encode a Certificate struct and write to w.
func (c *Certificate) Encode(w io.Writer) error {
	if err := c.EncodeHashable(w); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.Hash); err != nil {
		return err
	}

	return nil
}

// Decode a Certificate struct from r into c.
func (c *Certificate) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &c.Signature); err != nil {
		return err
	}

	lSteps, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	c.Steps = make([]*Step, lSteps)
	for i := uint64(0); i < lSteps; i++ {
		c.Steps[i] = &Step{}
		if err := encoding.ReadUint32(r, binary.LittleEndian, &c.Steps[i].StepNum); err != nil {
			return err
		}

		lData, err := encoding.ReadVarInt(r)
		if err != nil {
			return err
		}

		c.Steps[i].Data = make([]*CertData, lData)
		for j := uint64(0); j < lData; j++ {
			c.Steps[i].Data[j] = &CertData{}
			if err := encoding.Read256(r, &c.Steps[i].Data[j].PubAddress); err != nil {
				return err
			}

			if err := encoding.Read256(r, &c.Steps[i].Data[j].Proof); err != nil {
				return err
			}
		}
	}

	if err := encoding.Read256(r, &c.Hash); err != nil {
		return err
	}

	return nil
}

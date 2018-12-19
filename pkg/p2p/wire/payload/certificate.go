package payload

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

type CertData struct {
	PubAddress []byte // 32 bytes
	Proof      []byte // 32 bytes
}

type Step struct {
	StepNum uint32
	Data    []*CertData
}

type Certificate struct {
	Signature []byte // Batched BLS signatures (32 bytes)
	Steps     []*Step
	Hash      []byte // Certificate hash (32 bytes)
}

func NewCertificate(sig []byte) *Certificate {
	return &Certificate{
		Signature: sig,
	}
}

func (c *Certificate) AddStep(step *Step) {
	c.Steps = append(c.Steps, step)
}

func NewStep(n uint32) *Step {
	return &Step{
		StepNum: n,
	}
}

func (s *Step) AddData(pubAddr, proof []byte) {
	data := &CertData{
		PubAddress: pubAddr,
		Proof:      proof,
	}

	s.Data = append(s.Data, data)
}

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

func (c *Certificate) Encode(w io.Writer) error {
	if err := c.EncodeHashable(w); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.Hash); err != nil {
		return err
	}

	return nil
}

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

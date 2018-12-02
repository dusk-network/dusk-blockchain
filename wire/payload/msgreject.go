package payload

import (
	"errors"
	"fmt"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
)

// RejectCode defines the known reject codes of the protocol.
type RejectCode uint8

// Reject codes
const (
	RejectMalformed RejectCode = 0x01
	RejectInvalid   RejectCode = 0x10
	RejectObsolete  RejectCode = 0x11
	RejectDuplicate RejectCode = 0x12
)

// MsgReject defines a reject message on the Dusk wire protocol.
type MsgReject struct {
	Message    string
	RejectCode RejectCode
	Reason     string
	Data       []byte
}

// NewMsgReject returns a new MsgReject, populated with the specified arguments.
func NewMsgReject(msg string, code RejectCode, reason string) *MsgReject {
	return &MsgReject{
		Message:    msg,
		RejectCode: code,
		Reason:     reason,
	}
}

// AddData will add data to the reject message if it's relevant.
// This function is only used in case of a transaction or block rejection,
// so it will always be 32 bytes.
func (m *MsgReject) AddData(hash []byte) error {
	if len(hash) != 32 {
		return errors.New("reject data needs to be a hash (32 bytes)")
	}

	m.Data = hash
	return nil
}

// Encode a MsgReject struct and write to w.
// Implements payload interface.
func (m *MsgReject) Encode(w io.Writer) error {
	if err := encoding.WriteString(w, m.Message); err != nil {
		return err
	}

	if err := encoding.PutUint8(w, uint8(m.RejectCode)); err != nil {
		return err
	}

	if err := encoding.WriteString(w, m.Reason); err != nil {
		return err
	}

	if m.Data != nil {
		if err := encoding.WriteHash(w, m.Data); err != nil {
			return err
		}
	}

	return nil
}

// Decode a MsgReject from r.
// Implements payload interface.
func (m *MsgReject) Decode(r io.Reader) error {
	msg, err := encoding.ReadString(r)
	if err != nil {
		return err
	}

	code, err := encoding.Uint8(r)
	if err != nil {
		return err
	}

	if RejectCode(code) != RejectMalformed &&
		RejectCode(code) != RejectInvalid &&
		RejectCode(code) != RejectObsolete &&
		RejectCode(code) != RejectDuplicate {
		return fmt.Errorf("invalid reject code %v", code)
	}

	reason, err := encoding.ReadString(r)
	if err != nil {
		return err
	}

	data, err := encoding.ReadHash(r)
	if err != nil {
		// If we get an EOF error, there was no data left in the reader,
		// and we can simply discard the error.
		if err != io.EOF {
			return err
		}
	}

	if data != nil {
		m.Data = data
	}

	m.Message = msg
	m.RejectCode = RejectCode(code)
	m.Reason = reason
	return nil
}

// Command returns the command string associated with the reject message.
// Implements payload interface.
func (m *MsgReject) Command() commands.Cmd {
	return commands.Reject
}

package util

import (
	"encoding/hex"
	"strings"
)

// StringifyBytes returns a semi readable representation of a byte array
func StringifyBytes(b []byte) string {
	var sb strings.Builder
	if b != nil && len(b) > 0 {
		sb.WriteString(hex.EncodeToString(b[0:5]))
		sb.WriteString("...")
		sb.WriteString(hex.EncodeToString(b[len(b)-5:]))
	} else {
		sb.WriteString("empty")
	}
	return sb.String()
}

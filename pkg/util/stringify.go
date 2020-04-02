package util

import (
	"encoding/hex"
	"strings"
)

// StringifyBytes returns a semi readable representation of a byte array
func StringifyBytes(b []byte) string {
	var sb strings.Builder
	if b != nil && len(b) > 0 {
		_, _ = sb.WriteString(hex.EncodeToString(b[0:5]))
		_, _ = sb.WriteString("...")
		_, _ = sb.WriteString(hex.EncodeToString(b[len(b)-5:]))
	} else {
		_, _ = sb.WriteString("empty")
	}
	return sb.String()
}

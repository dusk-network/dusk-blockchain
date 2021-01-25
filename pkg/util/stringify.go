// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package util

import (
	"encoding/hex"
	"strings"
)

// StringifyBytes returns a semi readable representation of a byte array.
func StringifyBytes(b []byte) string {
	var sb strings.Builder

	if len(b) > 0 {
		_, _ = sb.WriteString(hex.EncodeToString(b[0:5]))
		_, _ = sb.WriteString("...")
		_, _ = sb.WriteString(hex.EncodeToString(b[len(b)-5:]))
	} else {
		_, _ = sb.WriteString("<empty>")
	}

	return sb.String()
}

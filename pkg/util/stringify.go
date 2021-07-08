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
		offset := 5
		if len(b) < offset {
			offset = len(b)
		}

		_, _ = sb.WriteString(hex.EncodeToString(b[0:offset]))

		if len(b)-offset > 0 {
			_, _ = sb.WriteString("...")
			_, _ = sb.WriteString(hex.EncodeToString(b[len(b)-offset:]))
		}
	} else {
		_, _ = sb.WriteString("<empty>")
	}

	return sb.String()
}

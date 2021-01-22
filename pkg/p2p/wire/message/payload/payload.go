// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package payload

// Safe is used for protection from race conditions.
type Safe interface {
	// Copy performs a deep copy of an object.
	Copy() Safe
}

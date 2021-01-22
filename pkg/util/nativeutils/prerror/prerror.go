// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package prerror

// Priority is the priority High,Medium,Low.
type Priority int

//nolint:golint
type PrError struct {
	Priority
	Err error
}

//nolint:golint
func (e *PrError) Error() string {
	return e.Err.Error()
}

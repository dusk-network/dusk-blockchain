// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package ruskmock

// Config contains a list of settings that determine the behavior
// of the `Server`.
type Config struct {
	PassScoreValidation           bool
	PassTransactionValidation     bool
	PassStateTransition           bool
	PassStateTransitionValidation bool
}

// DefaultConfig returns the default configuration for the Rusk mock server.
func DefaultConfig() *Config {
	return &Config{
		PassScoreValidation:           true,
		PassTransactionValidation:     true,
		PassStateTransition:           true,
		PassStateTransitionValidation: true,
	}
}

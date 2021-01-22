// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package diagnostics

import log "github.com/sirupsen/logrus"

// LogPublishErrors will log errors for a message. This method is mostly used for debug messages from rpcBus.Publish return errList.
func LogPublishErrors(msg string, errorList []error) {
	for _, err := range errorList {
		log.WithError(err).Debug(msg)
	}
}

// LogError logs a single error it uses LogPublishErrors in the background.
func LogError(msg string, err error) {
	LogPublishErrors(msg, []error{err})
}

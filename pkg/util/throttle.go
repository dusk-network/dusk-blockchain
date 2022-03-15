// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package util

import (
	"errors"
	"time"
)

// Throttle sleeps for up to maxDelayMilli but subtracting the time passed since startTimeMilli.
func Throttle(startTimeMilli int64, maxDelayMilli int64) (time.Duration, error) {
	if startTimeMilli == 0 || maxDelayMilli == 0 {
		return time.Duration(0), errors.New("invalid input")
	}

	delta := time.Now().UnixMilli() - startTimeMilli

	if delta > 0 && delta < maxDelayMilli {
		duration := time.Duration((maxDelayMilli - delta) * int64(time.Millisecond))

		if duration > 0 {
			time.Sleep(duration)
			return duration, nil
		}
	}

	return time.Duration(0), errors.New("max delay exceeded")
}

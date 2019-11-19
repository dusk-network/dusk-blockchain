package selection

import "testing"

func TestStopNilTimer(t *testing.T) {
	timer := &timer{}
	timer.stop()
}

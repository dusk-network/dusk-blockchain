package logger_test

import (
	"bytes"
	"encoding/json"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon/logger"
)

var withTimeTest = []struct {
	fields   log.Fields
	msg      string
	expected []string
}{
	{nil, "pippo", []string{"time", "process"}},
	{log.Fields{"topolino": "pluto"}, "pippo", []string{"time", "process", "topolino"}},
}

func TestWithTime(t *testing.T) {
	for _, tt := range withTimeTest {
		// setup
		logBase, b, data := setup(nil)

		// tested function
		logBase.WithTime(tt.fields).Info(tt.msg)
		assert.NoError(t, json.Unmarshal(b.Bytes(), &data))

		// checking msg
		assert.Equal(t, tt.msg, data["msg"])
		// checking that the fields are there
		for _, exp := range tt.expected {
			_, ok := data[exp]
			assert.True(t, ok)
		}
	}
}

func setup(formatter log.Formatter) (*logger.LogBase, *bytes.Buffer, map[string]interface{}) {
	var data map[string]interface{}
	b := new(bytes.Buffer)
	logBase := logger.New(b, formatter)
	return logBase, b, data
}

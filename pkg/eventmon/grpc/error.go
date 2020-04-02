package grpc

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pb "github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	log "github.com/sirupsen/logrus"
)

//NotifyError opens a connection to the monitoring server any time there
//is an error happening within the node
func (c *Client) NotifyError(parent context.Context, alert *pb.ErrorAlert) error {
	return c.send(parent, func(mon pb.MonitorClient, ctx context.Context) error {
		lg.WithField("log", alert.Msg).Debugln("notifying error to monitoring client")
		_, err := mon.NotifyError(ctx, alert)
		lg.Debugln("error notified")
		return err
	})
}

// ConvertToAlert converts a log.Entry into a monitor.ErrorAlert. It is done
// outside NotifyError function to not incur in locking problems with logrus
// trying to use the entry while it gets processed in the supervisor's loop
func ConvertToAlert(entry *log.Entry) *pb.ErrorAlert {
	alert := &pb.ErrorAlert{
		Level:           convertLevel(entry.Level),
		Msg:             entry.Message,
		TimestampMillis: entry.Time.Format(time.StampMilli),
		Fields:          convertFields(entry.Data),
	}

	if entry.Caller != nil {
		alert.File = entry.Caller.File
		alert.Line = uint32(entry.Caller.Line)
		alert.Function = entry.Caller.Function
	}
	return alert
}

func convertFields(fields log.Fields) []*pb.Field {
	converted := make([]*pb.Field, 0)
	for name, value := range fields {
		var strVal string
		switch value.(type) {
		case string:
			strVal = value.(string)
		case int:
			strVal = fmt.Sprintf("%d", int64(value.(int)))
		case int8:
			strVal = fmt.Sprintf("%d", value.(int8))
		case int16:
			strVal = fmt.Sprintf("%d", value.(int16))
		case int32:
			strVal = fmt.Sprintf("%d", value.(int32))
		case int64:
			strVal = fmt.Sprintf("%d", value.(int64))
		case uint:
			strVal = fmt.Sprintf("%d", value.(uint))
		case uint8:
			strVal = fmt.Sprintf("%d", value.(uint8))
		case uint16:
			strVal = fmt.Sprintf("%d", value.(uint16))
		case uint32:
			strVal = fmt.Sprintf("%d", value.(uint32))
		case uint64:
			strVal = fmt.Sprintf("%d", value.(uint64))
		case bool:
			strVal = strconv.FormatBool(value.(bool))
		case []byte:
			strVal = string(value.([]byte))
		case error:
			err := value.(error)
			strVal = fmt.Sprintf("%v", err)
		default:
			continue
		}

		field := &pb.Field{
			Field: name,
			Value: strVal,
		}
		converted = append(converted, field)
	}
	return converted
}

func convertLevel(l log.Level) pb.Level {
	switch l {
	case log.WarnLevel:
		return pb.Level_WARN
	case log.ErrorLevel:
		return pb.Level_ERROR
	case log.FatalLevel:
		return pb.Level_FATAL
	case log.PanicLevel:
		return pb.Level_PANIC
	}
	return pb.Level_ERROR
}

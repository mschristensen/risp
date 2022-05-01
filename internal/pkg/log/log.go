// Package log add logging utilities.
package log

import (
	"strings"
	"time"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var logger logrus.FieldLogger = logrus.StandardLogger()

// SetLogger sets the default logger's level.
func SetLogger(level string) {
	logrus.SetLevel(logrus.ErrorLevel)
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = time.RFC3339
	logrus.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	switch strings.ToLower(level) {
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.ErrorLevel)
	}
}

func ClientMessageToFields(msg *risppb.ClientMessage) logrus.Fields {
	id, err := uuid.FromBytes(msg.Uuid)
	if err != nil {
		logger.Fatalln(err)
	}
	return logrus.Fields{
		"uuid":   id.String(),
		"state":  msg.State.String(),
		"ack":    msg.Ack,
		"len":    msg.Len,
		"window": msg.Window,
	}
}

func ServerMessageToFields(msg *risppb.ServerMessage) logrus.Fields {
	return logrus.Fields{
		"state":    msg.State.String(),
		"index":    msg.Index,
		"payload":  msg.Payload,
		"checksum": msg.Checksum,
	}
}

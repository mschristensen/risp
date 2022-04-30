// Package log add logging utilities.
package log

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

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

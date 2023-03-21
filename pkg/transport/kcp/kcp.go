package kcp

import (
	"log"
)

type LoggingLevel int

const (
	LoggingLevelNone LoggingLevel = iota
	LoggingLevelError
	LoggingLevelDebug
	LoggingLevelTrace
)

var (
	loggingLevel LoggingLevel = LoggingLevelError
)

func SetLoggingLevel(level LoggingLevel) {
	loggingLevel = level
}

func (l LoggingLevel) string() string {
	switch l {
	default:
	case LoggingLevelNone:
	case LoggingLevelError:
		return "[ERROR] kcp: "
	case LoggingLevelDebug:
		return "[DEBUG] kcp: "
	}
	return ""
}

func (l LoggingLevel) enabled(level LoggingLevel) bool {
	return l >= level
}

func logging(level LoggingLevel, format string, args ...any) {
	if loggingLevel.enabled(level) {
		log.Printf(level.string()+format, args)
	}
}

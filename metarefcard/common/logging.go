package common

import (
	"fmt"
	"log"
)

type handler func(*LogEntry)

var logHandler handler = nil

// RegisterHandler stores the log handler to send logging messages to
func RegisterHandler(handler handler) {
	logHandler = handler
}

// DbgMsg prints an informational message
func DbgMsg(format string, v ...interface{}) {
	log.Println(fmt.Sprintf(format, v...))
}

// LogMsg logs an informational message
func LogMsg(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Println(msg)
	if logHandler != nil {
		logHandler(&LogEntry{false, msg})
	}
}

// LogErr logs an error message
func LogErr(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Println(fmt.Sprintf("Error: %s", msg))
	if logHandler != nil {
		logHandler(&LogEntry{true, msg})
	}
}

// LogEntry contains the message and metadata
type LogEntry struct {
	IsError bool
	Msg     string
}

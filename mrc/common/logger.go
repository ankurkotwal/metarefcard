package common

import (
	"fmt"
	"log"
)

// Logger is a log utility to log to
type Logger []*LogEntry

// Dbg prints an informational message
func (l *Logger) Dbg(format string, v ...interface{}) {
	log.Printf("%s\n", fmt.Sprintf(format, v...))
}

// Msg logs an informational message
func (l *Logger) Msg(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Println(msg)
	*l = append(*l, &LogEntry{false, msg})
}

// Err logs an error message
func (l *Logger) Err(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("%s\n", fmt.Sprintf("Error: %s", msg))
	*l = append(*l, &LogEntry{true, msg})
}

// Fatal calls log.Fatalf
func (l *Logger) Fatal(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// NewLog creates a new logger
func NewLog() *Logger {
	return new(Logger)
}

// LogEntry contains the message and metadata
type LogEntry struct {
	IsError bool
	Msg     string
}

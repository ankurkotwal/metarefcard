package common

import (
	"fmt"
	"log"
	"sync"
)

// Logger is a log utility to log to
type Logger struct {
	Entries   []*LogEntry
	FatalFunc func(format string, v ...interface{})
	mu        sync.Mutex
}

// Dbg prints an informational message
func (l *Logger) Dbg(format string, v ...interface{}) {
	log.Printf("%s\n", fmt.Sprintf(format, v...))
}

// Msg logs an informational message
func (l *Logger) Msg(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Println(msg)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Entries = append(l.Entries, &LogEntry{false, msg})
}

// Err logs an error message
func (l *Logger) Err(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("%s\n", fmt.Sprintf("Error: %s", msg))
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Entries = append(l.Entries, &LogEntry{true, msg})
}

// Fatal calls log.Fatalf
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.FatalFunc(format, v...)
}

// NewLog creates a new logger
func NewLog() *Logger {
	return &Logger{
		Entries:   make([]*LogEntry, 0),
		FatalFunc: log.Fatalf,
	}
}

// LogEntry contains the message and metadata
type LogEntry struct {
	IsError bool
	Msg     string
}

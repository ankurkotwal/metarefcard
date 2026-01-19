package common

import (
	"os"
	"os/exec"
	"testing"
)

func TestMsg(t *testing.T) {
	l := NewLog()
	msg := "test message"
	l.Msg("%s", msg)

	if len(*l) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(*l))
	}
	if (*l)[0].Msg != msg {
		t.Errorf("Expected message '%s', got '%s'", msg, (*l)[0].Msg)
	}
	if (*l)[0].IsError {
		t.Errorf("Expected IsError false, got true")
	}
}

func TestFatal(t *testing.T) {
	// Re-run the test in a subprocess
	if os.Getenv("BE_CRASHER") == "1" {
		l := NewLog()
		l.Fatal("fatal error")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

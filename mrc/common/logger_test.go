package common

import (
	"os"
	"os/exec"
	"testing"
)

func TestLogger_Msg(t *testing.T) {
	log := NewLog()
	log.Msg("Test message %s", "arg")

	if len(*log) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(*log))
	}
	if (*log)[0].IsError {
		t.Error("Expected IsError false")
	}
	if (*log)[0].Msg != "Test message arg" {
		t.Errorf("Unexpected message: %s", (*log)[0].Msg)
	}
}

func TestLogger_Fatal(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		log := NewLog()
		log.Fatal("Fatal error")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestLogger_Fatal")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

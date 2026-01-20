package common

import "testing"

func TestLogger_Msg(t *testing.T) {
	log := NewLog()
	log.Msg("test message %d", 1)
	
	if len(log.Entries) != 1 {
		t.Error("Expected 1 entry")
	}
	if log.Entries[0].Msg != "test message 1" {
		t.Errorf("Wrong message: %s", log.Entries[0].Msg)
	}
	if log.Entries[0].IsError {
		t.Error("Expected not error")
	}
}

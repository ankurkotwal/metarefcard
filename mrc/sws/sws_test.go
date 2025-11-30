package sws

import (
	"strings"
	"testing"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

func TestGetGameInfo(t *testing.T) {
	label, description, handler, matcher := GetGameInfo()

	if label != "sws" {
		t.Errorf("Expected label 'sws', got '%s'", label)
	}
	if !strings.Contains(description, "Star Wars Squadrons") {
		t.Errorf("Expected description to contain 'Star Wars Squadrons', got '%s'", description)
	}
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
	if matcher == nil {
		t.Error("Expected non-nil matcher")
	}
}

func TestMatchGameInputToModel(t *testing.T) {
	// SWS matchGameInputToModel just passes through the input, unlike fs2020 which parses regexes.
	// The parsing happens in loadInputFiles for SWS.
	// So for this test we verify the pass-through behavior.

	log := common.NewLog()
	deviceInputs := make(common.DeviceInputs)
	gameInputMap := make(common.InputTypeMapping)

	tests := []struct {
		name          string
		actionData    common.GameInput
		expectedPrimary string
		expectedCount int
	}{
		{
			name:          "Pass Through Button",
			actionData:    common.GameInput{"Button 1", ""},
			expectedPrimary: "Button 1",
			expectedCount: 2, // It returns GameInput which is []string of length 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := matchGameInputToModel("TestDevice", tt.actionData, deviceInputs, gameInputMap, log)
			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(result))
			}
			if len(result) > 0 && result[common.InputPrimary] != tt.expectedPrimary {
				t.Errorf("Expected primary match %s, got %s", tt.expectedPrimary, result[common.InputPrimary])
			}
		})
	}
}

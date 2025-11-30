package fs2020

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

func TestGetGameInfo(t *testing.T) {
	label, description, handler, matcher := GetGameInfo()

	if label != "fs2020" {
		t.Errorf("Expected label 'fs2020', got '%s'", label)
	}
	if !strings.Contains(description, "Flight Simulator 2020") {
		t.Errorf("Expected description to contain 'Flight Simulator 2020', got '%s'", description)
	}
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
	if matcher == nil {
		t.Error("Expected non-nil matcher")
	}
}

func TestMatchGameInputToModel(t *testing.T) {
	// Initialize sharedRegexes
	// In the real code this is done in handleRequest via sync.Once, but for unit testing matchGameInputToModel
	// we need to set it up manually or call handleRequest once.
	// Since handleRequest needs config files, we might just manually initialize the regexes for testing.

	// From config/fs2020.yaml (we should ideally read it but for unit test we can mock)
	// Regexes:
	//   Button: "Joystick Button ([0-9]+)"
	//   Axis: "Joystick ([L,R])-Axis ([X,Y,Z])"
	//   Pov: "Joystick POV( [0-9])* ([a-zA-Z]+)"
	//   Rotation: "Joystick R-Axis ([X,Y,Z])"
	//   Slider: "Joystick Slider ([0-9]+)"

	sharedRegexes.Button = regexp.MustCompile(`Joystick Button ([0-9]+)`)
	sharedRegexes.Axis = regexp.MustCompile(`Joystick ([L,R])-Axis ([X,Y,Z])`)
	sharedRegexes.Pov = regexp.MustCompile(`Joystick POV( [0-9])* ([a-zA-Z]+)`)
	sharedRegexes.Rotation = regexp.MustCompile(`Joystick R-Axis ([X,Y,Z])`)
	sharedRegexes.Slider = regexp.MustCompile(`Joystick Slider ([0-9]+)`)

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
			name:          "Button Match",
			actionData:    common.GameInput{"Joystick Button 1", ""},
			expectedPrimary: "1",
			expectedCount: 1,
		},
		{
			name:          "Axis Match",
			actionData:    common.GameInput{"Joystick L-Axis X", ""},
			expectedPrimary: "LXAxis",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := matchGameInputToModel("TestDevice", tt.actionData, deviceInputs, gameInputMap, log)
			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(result))
			}
			if len(result) > 0 && result[0] != tt.expectedPrimary {
				t.Errorf("Expected primary match %s, got %s", tt.expectedPrimary, result[0])
			}
		})
	}
}

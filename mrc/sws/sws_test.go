package sws

import (
	"os"
	"path/filepath"
	"regexp"
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

func TestLoadInputFiles(t *testing.T) {
	// Initialize regexes
	log := common.NewLog()
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/sws.yaml")
	gameData := common.LoadGameModel(configPath, "SWS Data", false, log)
	sharedGameData = gameData
	
	sharedRegexes = swsRegexes{
		Bind:     regexp.MustCompile(sharedGameData.Regexes["Bind"]),
		Joystick: regexp.MustCompile(sharedGameData.Regexes["Joystick"]),
	}

	deviceMap := common.DeviceNameFullToShort{
		"Saitek Pro Flight X-55 Rhino Stick":    "SaitekX55Joystick",
		"Saitek Pro Flight X-55 Rhino Throttle": "SaitekX55Throttle",
	}

	// Read sample file
	testDataPath := "../../testdata/sws/Saitek_Pro_Flight_X-55_Rhino.profile"
	fileContent, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Fatalf("Failed to read test data file: %v", err)
	}

	files := [][]byte{fileContent}

	// Mocking config flags
	gameBinds, deviceNames, contexts := loadInputFiles(files, deviceMap, log, true, true)

	if len(gameBinds) == 0 {
		t.Error("Expected game binds to be populated")
	}

	if !deviceNames["SaitekX55Joystick"] {
		t.Error("Expected SaitekX55Joystick in deviceNames")
	}
	
	if len(contexts) == 0 {
		t.Error("Expected contexts to be populated")
	}
}

func TestInterpretInput(t *testing.T) {
	log := common.NewLog()
	
	// Test case 1: Axis 8 on Throttle -> XAxis
	details := &swsActionDetails{
		Axis:     "8",
		DeviceID: "1", // deviceId isn't really used inside interpretInput logic shown in sws.go (but passed in args)
		// Actually device string is passed separately
	}
	
	got, err := interpretInput(details, "SaitekX55Throttle", "TestContext", "TestAction", log)
	if err != nil {
		t.Errorf("interpretInput failed: %v", err)
	}
	if got != "XAxis" {
		t.Errorf("interpretInput = %v, want XAxis", got)
	}

	// Test case 2: Button 46 on Stick -> RZAxis (Rotation)
	// Button 46 falls through the hardcoded ranges in interpretInput and hits the device-specific logic.
	
	detailsButton := &swsActionDetails{
		Axis:     "26", // 26 triggers button logic in existing code
		Button:   "46",
		DeviceID: "0",
	}

	got, err = interpretInput(detailsButton, "SaitekX55Joystick", "TestContext", "TestAction", log)
	if err != nil {
		t.Errorf("interpretInput failed: %v", err)
	}
	if got != "RZAxis" {
		t.Errorf("interpretInput = %v, want RZAxis", got)
	}
}

func TestLoadInputFiles_CorruptData(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{}

	// Random garbage data
	corruptFile := []byte(`
		This is not a valid line
		GstInput.JoystickDevice1 but incomplete...
		Just random text
	`)
	
	files := [][]byte{corruptFile}

	// Should not panic, just ignore
	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, true)
	
	if len(gameBinds[common.ProfileDefault]) > 0 {
		t.Errorf("Expected empty gameBinds for corrupt data, got %v", gameBinds)
	}
}

func TestLoadInputFiles_ErroneousData(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"Saitek Pro Flight X-55 Rhino Stick": "SaitekX55Joystick",
	}

	// Valid format but unknown device
	unknownDeviceData := []byte(`
		GstInput.JoystickDevice1 Unknown Joystick
		GstKeyBinding.IncomDefaultInputConcepts.ConceptActivate.1.button 5
		GstKeyBinding.IncomDefaultInputConcepts.ConceptActivate.1.deviceid 0
	`)

	files := [][]byte{unknownDeviceData}

	// loadInputFiles should see "Unknown Joystick", fail to map it in deviceMap, and log error/skip it.
	// Subsequently, binds referring to deviceid 0 (which maps to joystick 1 -> Unknown) should be skipped.

	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, true)

	if len(gameBinds[common.ProfileDefault]) != 0 {
		// Because device 1 was unknown, it shouldn't be in the index, 
		// so actions for deviceid 0 should be skipped.
		t.Errorf("Expected gameBinds to be empty for unknown device, got %v", gameBinds)
	}
}

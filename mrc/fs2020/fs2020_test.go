package fs2020

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

func TestHandleRequest(t *testing.T) {
	configPath = "../../config/fs2020.yaml"
	config := &common.Config{
		AlternateColours: []string{"#FFFFFF"},
		Devices: common.Devices{
			DeviceToShortNameMap: map[string]string{
				"Alpha Flight Controls": "AlphaFlightControls",
			},
		},
		DebugOutput:   true,
		VerboseOutput: true,
	}
	log := common.NewLog()

	// Generic valid XML
	validXML := []byte(`<Device DeviceName="Alpha Flight Controls"><Context ContextName="PLANE"><Action ActionName="KEY_AP_MASTER"><Primary><KEY Information="Button 4"/></Primary></Action></Context></Device>`)
	files := [][]byte{validXML}

	_, gameBinds, neededDevices, contextColours, logo := handleRequest(files, config, log)

	if len(gameBinds) == 0 {
		t.Error("Expected game binds")
	}
	if !neededDevices["AlphaFlightControls"] {
		t.Error("Expected AlphaFlightControls in neededDevices")
	}
	if contextColours == nil {
		t.Error("Expected contextColours")
	}
	if logo != "fs2020" {
		t.Errorf("Expected logo 'fs2020', got '%s'", logo)
	}
}

func TestLoadInputFiles(t *testing.T) {
	// Setup generic config for testing
	log := common.NewLog()
	
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
		"T.A320 Pilot": "T-A320Pilot",
	}

	// Read a sample file from testdata
	// Assuming running from package dir
	testDataPath := "../../testdata/fs2020/Alpha_Flight_Controls.xml"
	fileContent, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Fatalf("Failed to read test data file: %v", err)
	}

	files := [][]byte{fileContent}

	gameBinds, neededDevices, contextColours := loadInputFiles(files, deviceMap, log, true, true)

	if len(gameBinds) == 0 {
		t.Error("Expected game binds to be populated")
	}
	
	if !neededDevices["AlphaFlightControls"] {
		t.Error("Expected AlphaFlightControls in neededDevices")
	}

	if len(contextColours) == 0 {
		t.Log("Contexts might be empty if not defined in the XML")
	}
}

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
	// Load config to populate regexes (Setup from HEAD)
	wd, _ := os.Getwd()
	// config is at ../../config/fs2020.yaml relative to package
	configPath := filepath.Join(wd, "../../config/fs2020.yaml")
	
	log := common.NewLog()
	
	// Load game data to get regex strings
	gameData, _ := common.LoadGameModel(configPath, "FS2020 Data", false, log)
	sharedGameData = gameData
	
	// Compile regexes manually as they are in fs2020.go
	sharedRegexes = fs2020Regexes{
		Button:   regexp.MustCompile(sharedGameData.Regexes["Button"]),
		Axis:     regexp.MustCompile(sharedGameData.Regexes["Axis"]),
		Pov:      regexp.MustCompile(sharedGameData.Regexes["Pov"]),
		Rotation: regexp.MustCompile(sharedGameData.Regexes["Rotation"]),
		Slider:   regexp.MustCompile(sharedGameData.Regexes["Slider"]),
	}

	deviceInputs := make(common.DeviceInputs)
	gameInputMap := make(common.InputTypeMapping)

	// Test cases (Adapted from HEAD to match Incoming's structure)
	tests := []struct {
		name          string
		actionData    common.GameInput
		deviceName    string
		expectedPrimary string
		expectedCount int
	}{
		{
			name:          "Standard Button",
			actionData:    common.GameInput{"Button 1", ""},
			deviceName:    "TestDevice",
			expectedPrimary: "1",
			expectedCount: 1,
		},
		{
			name:          "Joystick Axis",
			actionData:    common.GameInput{"Axis X", ""}, // Adjusted input to match regex
			deviceName:    "TestDevice",
			expectedPrimary: "XAxis",
			expectedCount: 1,
		},
		{
			name:          "POV Hat Up",
			actionData:    common.GameInput{"POV1_UP", ""},
			deviceName:    "TestDevice",
			expectedPrimary: "POV1Up",
			expectedCount: 1,
		},
		{
			name:            "Rotation Z",
			actionData:      common.GameInput{"Rotation Z", ""},
			deviceName:      "TestDevice",
			expectedPrimary: "RZAxis",
			expectedCount:   1,
		},
		{
			name:            "Slider X (Unmapped)",
			actionData:      common.GameInput{"Slider X", ""},
			deviceName:      "TestDevice",
			expectedPrimary: "", // Error logged, empty return
			expectedCount:   0,
		},
		{
			name:            "Slider X (Mapped)",
			actionData:      common.GameInput{"Slider X", ""},
			deviceName:      "MappedDevice",
			expectedPrimary: "UAxis",
			expectedCount:   1,
		},
		{
			name:            "Axis X (Mapped)",
			actionData:      common.GameInput{"Axis X", ""},
			deviceName:      "MappedDevice",
			expectedPrimary: "UAxis",
			expectedCount:   1,
		},
	}
	
	// Mock input map for mapped device
	gameInputMap["Slider"] = map[string]string{"X": "U"}
	gameInputMap["Axis"] = map[string]string{"X": "U"} // Map X to U

	// Mock valid inputs on the device
	deviceInputs["UAxis"] = common.InputData{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Select input map based on device name for testing purposes
			var currentInputMap common.InputTypeMapping
			if tt.deviceName == "MappedDevice" {
				currentInputMap = gameInputMap
			}

			result, _ := matchGameInputToModel(tt.deviceName, tt.actionData, deviceInputs, currentInputMap, log)
			// matchGameInputToModel returns (common.GameInput, string). GameInput is []string
			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(result))
			}
			if len(result) > 0 && result[common.InputPrimary] != tt.expectedPrimary {
				t.Errorf("Expected primary match %s, got %s", tt.expectedPrimary, result[common.InputPrimary])
			}
		})
	}
}

func TestLoadInputFiles_CorruptXML(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{}

	// Invalid XML content
	corruptFile := []byte(`<Device DeviceName="Alpha Flight Controls"><Context ContextName="PLANE">Unclosed Tag`)
	
	files := [][]byte{corruptFile}

	// Should not panic and ideally return empty/partial result
	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, true)
	
	if len(gameBinds[common.ProfileDefault]) > 0 {
		// Just ensuring it didn't crash. Empty result expected or partial.
		// Since we didn't define any Actions properly, it should likely be empty.
	}
}

func TestLoadInputFiles_ErroneousData(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
	}

	// Valid XML, but Unknown Device
	unknownDeviceXML := []byte(`
		<Device DeviceName="Unknown Device 123">
			<Context ContextName="PLANE">
				<Action ActionName="KEY_AP_MASTER">
					<Primary>
						<KEY Information="Button 4"/>
					</Primary>
				</Action>
			</Context>
		</Device>
	`)

	files := [][]byte{unknownDeviceXML}

	// Should handle gracefully (log error) and skip
	gameBinds, neededDevices, _ := loadInputFiles(files, deviceMap, log, true, true)

	if len(neededDevices) != 0 {
		t.Errorf("Expected neededDevices to be empty for unknown device, got %v", neededDevices)
	}

	if len(gameBinds[common.ProfileDefault]) != 0 {
		// With no valid devices, this should be empty
		t.Errorf("Expected gameBinds to be empty, got %v", gameBinds)
	}
}

func BenchmarkLoadInputFiles(b *testing.B) {
	// Setup generic config for testing
	log := common.NewLog()
	
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
	}

	// Read a sample file from testdata
	wd, _ := os.Getwd()
	testDataPath := filepath.Join(wd, "../../testdata/fs2020/Alpha_Flight_Controls.xml")
	fileContent, err := os.ReadFile(testDataPath)
	if err != nil {
		b.Fatalf("Failed to read test data file: %v", err)
	}

	files := [][]byte{fileContent}
	
	// Ensure regexes are inited
	configPath := filepath.Join(wd, "../../config/fs2020.yaml")
	gameData, _ := common.LoadGameModel(configPath, "FS2020 Data", false, log)
	sharedGameData = gameData
	sharedRegexes = fs2020Regexes{
		Button:   regexp.MustCompile(sharedGameData.Regexes["Button"]),
		Axis:     regexp.MustCompile(sharedGameData.Regexes["Axis"]),
		Pov:      regexp.MustCompile(sharedGameData.Regexes["Pov"]),
		Rotation: regexp.MustCompile(sharedGameData.Regexes["Rotation"]),
		Slider:   regexp.MustCompile(sharedGameData.Regexes["Slider"]),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loadInputFiles(files, deviceMap, log, false, false)
	}
}

func TestLoadInputFiles_WithProfile(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
	}

	// XML with FriendlyName (profile)
	xmlWithProfile := []byte(`
		<FriendlyName>Custom Profile</FriendlyName>
		<Device DeviceName="Alpha Flight Controls">
			<Context ContextName="PLANE">
				<Action ActionName="KEY_AP_MASTER">
					<Primary>
						<KEY Information="Button 1"/>
					</Primary>
				</Action>
			</Context>
		</Device>
	`)

	files := [][]byte{xmlWithProfile}
	gameBinds, neededDevices, _ := loadInputFiles(files, deviceMap, log, true, false)

	if !neededDevices["AlphaFlightControls"] {
		t.Error("Expected AlphaFlightControls in neededDevices")
	}

	// Check that Custom Profile exists
	if _, found := gameBinds["Custom Profile"]; !found {
		t.Error("Expected 'Custom Profile' in gameBinds")
	}
}

func TestLoadInputFiles_DuplicateContext(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
	}

	// XML with duplicate context names
	xmlDupContext := []byte(`
		<Device DeviceName="Alpha Flight Controls">
			<Context ContextName="PLANE">
				<Action ActionName="ACTION_1">
					<Primary><KEY Information="Button 1"/></Primary>
				</Action>
			</Context>
			<Context ContextName="PLANE">
				<Action ActionName="ACTION_2">
					<Primary><KEY Information="Button 2"/></Primary>
				</Action>
			</Context>
		</Device>
	`)

	files := [][]byte{xmlDupContext}
	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, false)

	// Should have logged an error for duplicate context but still processed
	if len(gameBinds) == 0 {
		t.Error("Expected gameBinds to be populated")
	}
}

func TestLoadInputFiles_DuplicateAction(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
	}

	// XML with duplicate action names
	xmlDupAction := []byte(`
		<Device DeviceName="Alpha Flight Controls">
			<Context ContextName="PLANE">
				<Action ActionName="ACTION_1">
					<Primary><KEY Information="Button 1"/></Primary>
				</Action>
				<Action ActionName="ACTION_1">
					<Primary><KEY Information="Button 2"/></Primary>
				</Action>
			</Context>
		</Device>
	`)

	files := [][]byte{xmlDupAction}
	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, false)

	// Should have logged an error for duplicate action but still processed
	if len(gameBinds) == 0 {
		t.Error("Expected gameBinds to be populated")
	}
}

func TestLoadInputFiles_WithSecondaryKey(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
	}

	// XML with secondary key
	xmlWithSecondary := []byte(`
		<Device DeviceName="Alpha Flight Controls">
			<Context ContextName="PLANE">
				<Action ActionName="ACTION_WITH_SECONDARY">
					<Primary><KEY Information="Button 1"/></Primary>
					<Secondary><KEY Information="Button 2"/></Secondary>
				</Action>
			</Context>
		</Device>
	`)

	files := [][]byte{xmlWithSecondary}
	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, false)

	// Verify secondary key was captured
	if actions, ok := gameBinds[common.ProfileDefault]["AlphaFlightControls"]["PLANE"]; ok {
		if action, ok := actions["ACTION_WITH_SECONDARY"]; ok {
			if action[common.InputSecondary] != "Button 2" {
				t.Errorf("Expected secondary 'Button 2', got '%s'", action[common.InputSecondary])
			}
		} else {
			t.Error("Expected ACTION_WITH_SECONDARY in actions")
		}
	} else {
		t.Error("Expected PLANE context in gameBinds")
	}
}

func TestMatchGameInputToModel_WithSecondary(t *testing.T) {
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/fs2020.yaml")
	log := common.NewLog()

	gameData, _ := common.LoadGameModel(configPath, "FS2020 Data", false, log)
	sharedGameData = gameData
	sharedRegexes = fs2020Regexes{
		Button:   regexp.MustCompile(sharedGameData.Regexes["Button"]),
		Axis:     regexp.MustCompile(sharedGameData.Regexes["Axis"]),
		Pov:      regexp.MustCompile(sharedGameData.Regexes["Pov"]),
		Rotation: regexp.MustCompile(sharedGameData.Regexes["Rotation"]),
		Slider:   regexp.MustCompile(sharedGameData.Regexes["Slider"]),
	}

	deviceInputs := make(common.DeviceInputs)

	// Test with both primary and secondary
	actionData := common.GameInput{"Button 1", "Button 2"}
	result, _ := matchGameInputToModel("TestDevice", actionData, deviceInputs, nil, log)

	if len(result) != 2 {
		t.Errorf("Expected 2 results for primary+secondary, got %d", len(result))
	}
	if len(result) >= 2 {
		if result[0] != "1" {
			t.Errorf("Expected primary '1', got '%s'", result[0])
		}
		if result[1] != "2" {
			t.Errorf("Expected secondary '2', got '%s'", result[1])
		}
	}
}

func TestMatchGameInputToModel_SecondaryNotMatched(t *testing.T) {
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/fs2020.yaml")
	log := common.NewLog()

	gameData, _ := common.LoadGameModel(configPath, "FS2020 Data", false, log)
	sharedGameData = gameData
	sharedRegexes = fs2020Regexes{
		Button:   regexp.MustCompile(sharedGameData.Regexes["Button"]),
		Axis:     regexp.MustCompile(sharedGameData.Regexes["Axis"]),
		Pov:      regexp.MustCompile(sharedGameData.Regexes["Pov"]),
		Rotation: regexp.MustCompile(sharedGameData.Regexes["Rotation"]),
		Slider:   regexp.MustCompile(sharedGameData.Regexes["Slider"]),
	}

	deviceInputs := make(common.DeviceInputs)

	// Test with primary matching but secondary not matching regex
	actionData := common.GameInput{"Button 1", "InvalidSecondary"}
	result, _ := matchGameInputToModel("TestDevice", actionData, deviceInputs, nil, log)

	// Should have only primary since secondary doesn't match
	if len(result) != 1 {
		t.Errorf("Expected 1 result (primary only), got %d", len(result))
	}
}

func TestMatchGameInputToModel_PrimaryNotMatched(t *testing.T) {
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/fs2020.yaml")
	log := common.NewLog()

	gameData, _ := common.LoadGameModel(configPath, "FS2020 Data", false, log)
	sharedGameData = gameData
	sharedRegexes = fs2020Regexes{
		Button:   regexp.MustCompile(sharedGameData.Regexes["Button"]),
		Axis:     regexp.MustCompile(sharedGameData.Regexes["Axis"]),
		Pov:      regexp.MustCompile(sharedGameData.Regexes["Pov"]),
		Rotation: regexp.MustCompile(sharedGameData.Regexes["Rotation"]),
		Slider:   regexp.MustCompile(sharedGameData.Regexes["Slider"]),
	}

	deviceInputs := make(common.DeviceInputs)

	// Test with primary not matching
	actionData := common.GameInput{"InvalidPrimary", ""}
	result, _ := matchGameInputToModel("TestDevice", actionData, deviceInputs, nil, log)

	// Should be empty since primary doesn't match
	if len(result) != 0 {
		t.Errorf("Expected 0 results for unmatched primary, got %d", len(result))
	}
}

func TestMatchGameInputToModelByRegex_POVWithNumber(t *testing.T) {
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/fs2020.yaml")
	log := common.NewLog()

	gameData, _ := common.LoadGameModel(configPath, "FS2020 Data", false, log)
	sharedGameData = gameData
	sharedRegexes = fs2020Regexes{
		Button:   regexp.MustCompile(sharedGameData.Regexes["Button"]),
		Axis:     regexp.MustCompile(sharedGameData.Regexes["Axis"]),
		Pov:      regexp.MustCompile(sharedGameData.Regexes["Pov"]),
		Rotation: regexp.MustCompile(sharedGameData.Regexes["Rotation"]),
		Slider:   regexp.MustCompile(sharedGameData.Regexes["Slider"]),
	}

	// Test POV with explicit number - regex is: (?i)Pov(\d?)[\s_]([[:alpha:]]+)
	// Input format should be like "Pov2_Up" or "POV2 Up"
	result := matchGameInputToModelByRegex("TestDevice", "POV2_Up", nil, nil, log)
	if result != "POV2Up" {
		t.Errorf("Expected POV2Up, got '%s'", result)
	}
}


func TestMatchGameInputToModelByRegex_RotationWithOverride(t *testing.T) {
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/fs2020.yaml")
	log := common.NewLog()

	gameData, _ := common.LoadGameModel(configPath, "FS2020 Data", false, log)
	sharedGameData = gameData
	sharedRegexes = fs2020Regexes{
		Button:   regexp.MustCompile(sharedGameData.Regexes["Button"]),
		Axis:     regexp.MustCompile(sharedGameData.Regexes["Axis"]),
		Pov:      regexp.MustCompile(sharedGameData.Regexes["Pov"]),
		Rotation: regexp.MustCompile(sharedGameData.Regexes["Rotation"]),
		Slider:   regexp.MustCompile(sharedGameData.Regexes["Slider"]),
	}

	// Test Rotation with override
	inputMap := common.InputTypeMapping{
		"Rotation": map[string]string{"Z": "U"},
	}
	result := matchGameInputToModelByRegex("TestDevice", "Rotation Z", nil, inputMap, log)
	if result != "UAxis" {
		t.Errorf("Expected UAxis (mapped from Z), got '%s'", result)
	}
}

func TestMatchGameInputToModelByRegex_SliderNotOnDevice(t *testing.T) {
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/fs2020.yaml")
	log := common.NewLog()

	gameData, _ := common.LoadGameModel(configPath, "FS2020 Data", false, log)
	sharedGameData = gameData
	sharedRegexes = fs2020Regexes{
		Button:   regexp.MustCompile(sharedGameData.Regexes["Button"]),
		Axis:     regexp.MustCompile(sharedGameData.Regexes["Axis"]),
		Pov:      regexp.MustCompile(sharedGameData.Regexes["Pov"]),
		Rotation: regexp.MustCompile(sharedGameData.Regexes["Rotation"]),
		Slider:   regexp.MustCompile(sharedGameData.Regexes["Slider"]),
	}

	// Test Slider mapped but not on device
	inputMap := common.InputTypeMapping{
		"Slider": map[string]string{"X": "U"},
	}
	deviceInputs := make(common.DeviceInputs)
	// UAxis not in deviceInputs

	result := matchGameInputToModelByRegex("TestDevice", "Slider X", deviceInputs, inputMap, log)
	if result != "" {
		t.Errorf("Expected empty result for slider not on device, got '%s'", result)
	}
}


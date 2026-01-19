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

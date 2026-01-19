package fs2020

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

func TestMatchGameInputToModelByRegex(t *testing.T) {
	// Load config to populate regexes
	wd, _ := os.Getwd()
	// config is at ../../config/fs2020.yaml relative to package
	configPath := filepath.Join(wd, "../../config/fs2020.yaml")
	
	log := common.NewLog()
	
	// Load game data to get regex strings
	gameData := common.LoadGameModel(configPath, "FS2020 Data", false, log)
	sharedGameData = gameData
	
	// Compile regexes manually as they are in fs2020.go
	sharedRegexes = fs2020Regexes{
		Button:   regexp.MustCompile(sharedGameData.Regexes["Button"]),
		Axis:     regexp.MustCompile(sharedGameData.Regexes["Axis"]),
		Pov:      regexp.MustCompile(sharedGameData.Regexes["Pov"]),
		Rotation: regexp.MustCompile(sharedGameData.Regexes["Rotation"]),
		Slider:   regexp.MustCompile(sharedGameData.Regexes["Slider"]),
	}
	
	// Test cases
	tests := []struct {
		name       string
		action     string
		deviceName string
		want       string
	}{
		{
			name:       "Standard Button",
			action:     "Button 1",
			deviceName: "TestDevice",
			want:       "1", 
		},
		{
			name:       "Joystick Axis",
			action:     "Axis X",
			deviceName: "TestDevice",
			want:       "XAxis", 
		},
		{
			name:       "POV Hat Up",
			action:     "POV1_UP",
			deviceName: "TestDevice",
			want:       "POV1Up",
		},
	}

	mockInputs := make(common.DeviceInputs)
	// We need to verify what matchGameInputToModelByRegex expects in inputs. 
	// required if looking for sliders?
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchGameInputToModelByRegex(tt.deviceName, tt.action, mockInputs, nil, log)
			if got != tt.want {
				t.Errorf("matchGameInputToModelByRegex() = %v, want %v", got, tt.want)
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
	gameData := common.LoadGameModel(configPath, "FS2020 Data", false, log)
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

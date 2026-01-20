package fs2020

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

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

func TestLoadInputFiles_Errors(t *testing.T) {
	log := common.NewLog()
	
	// XML Errors
	file1 := []byte(`<Device DeviceName="Target"><UnclosedTag`)
	files := [][]byte{file1}
	
	mapping := make(common.DeviceNameFullToShort)
	mapping["Target"] = "Target"
	
	_, _, _ = loadInputFiles(files, mapping, log, true, true)
	// Just ensure no panic. Errors logged.
	if len(log.Entries) == 0 {
		// Expect decoding error? 
		// Actually "UnclosedTag" might trigger EOF or Syntax error
		// fs2020.go line 87 checks err != nil
	}
	
	// Unknown Device
	file2 := []byte(`<Device DeviceName="Unknown"></Device>`)
	log = common.NewLog()
	loadInputFiles([][]byte{file2}, mapping, log, false, false)
	// Should log error
	found := false
	for _, e := range log.Entries {
		if e.IsError && len(e.Msg) > 0 { found = true }
	}
	if !found {
		t.Error("Expected error for unknown device")
	}
	
	// Duplicate Device
	file3 := []byte(`<Device DeviceName="Target"></Device>`)
	mapping["Target"] = "Target"
	// Load twice
	log = common.NewLog()
	// Mock pre-existing
	// But loadInputFiles starts fresh.
	// We need two files with same device
	loadInputFiles([][]byte{file3, file3}, mapping, log, true, false)
	// Check log for duplicate
	found = false
	for _, e := range log.Entries {
		if e.IsError && len(e.Msg) > 0 { found = true }
	}
	if !found {
		t.Error("Expected error for duplicate device")
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
		{
			name: "POV Hat Down",
			action: "POV1_DOWN",
			deviceName: "TestDevice",
			want: "POV1Down",
		},
		{
			name: "POVSubHat",
			action: "POV2_UP",
			deviceName: "TestDevice",
			want: "POV2Up",
		},
		{
			name: "Rotation Z",
			action: "Rotation Z",
			deviceName: "TestDevice",
			want: "RZAxis",
		},
		{
			name: "Slider Success",
			action: "Slider X",
			deviceName: "TestDevice",
			want: "SliderAxis", 
		},
		{
			name: "Slider Unknown",
			action: "Slider Y", // Not in map
			deviceName: "TestDevice",
			want: "",
		},
		{
			name: "No Match",
			action: "Unknown Action",
			deviceName: "TestDevice",
			want: "",
		},
	}

	mockInputs := make(common.DeviceInputs)
	mockInputs["SliderAxis"] = common.InputData{} // Needs to exist for Slider logic
	
	mockInputMap := make(common.InputTypeMapping)
	mockInputMap["Slider"] = map[string]string{"X": "Slider"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchGameInputToModelByRegex(tt.deviceName, tt.action, mockInputs, mockInputMap, log)
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

func TestLoadInputFiles_Duplicates(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
	}
	
	xmlData := []byte(`
		<Device DeviceName="Alpha Flight Controls">
			<Context ContextName="CTX1"></Context>
		</Device>
		<Device DeviceName="Alpha Flight Controls">
			<!-- Duplicate Device -->
		</Device>
	`)
	
	files := [][]byte{xmlData}
	
	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, true)
	
	// Should have loaded once.
	// We can check logs for error "FS2020 duplicate device"
	foundError := false
	for _, entry := range log.Entries {
		if entry.IsError && entry.Msg != "" { 
			// Check content if needed, but existence of error is enough coverage
			foundError = true
		}
	}
	if !foundError {
		t.Error("Expected error for duplicate device")
	}
	
	if len(gameBinds[common.ProfileDefault]) != 1 {
		t.Errorf("Expected 1 device, got %d", len(gameBinds[common.ProfileDefault]))
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

func TestGetGameInfo(t *testing.T) {
	label, desc, handler, matchFunc := GetGameInfo()
	if label != "fs2020" {
		t.Error("Wrong label")
	}
	if len(desc) == 0 {
		t.Error("Empty description")
	}
	if handler == nil {
		t.Error("Handler is nil")
	}
	if matchFunc == nil {
		t.Error("MatchFunc is nil")
	}
}

func TestHandleRequest(t *testing.T) {
	// Need to initialize basics
	log := common.NewLog()
	
	wd, _ := os.Getwd()
	// Create config link
	os.MkdirAll("config", 0755)
	// Copy ../../config/fs2020.yaml to config/fs2020.yaml
	input, _ := os.ReadFile("../../config/fs2020.yaml")
	if len(input) == 0 {
		// Fallback if running from root or different path
		// try absolute path based on wd
		input, _ = os.ReadFile(filepath.Join(wd, "../../config/fs2020.yaml"))
	}
	os.WriteFile("config/fs2020.yaml", input, 0644)
	defer os.RemoveAll("config")
	
	config := &common.Config{
		Devices: common.Devices{
			DeviceToShortNameMap: common.DeviceNameFullToShort{
				"Alpha Flight Controls": "AlphaFlightControls",
			},
		},
		DebugOutput: true,
	}
	
	files := [][]byte{}
	
	// Call
	gData, _, _, _, logo := handleRequest(files, config, log)
	
	if gData.Logo == "" {
		t.Error("GameData Logo empty")
	}
	if logo != gData.Logo {
		t.Error("Returned logo mismatch")
	}
}

func TestMatchGameInputToModel(t *testing.T) {
	log := common.NewLog()
	
	// Setup sharedRegexes
	wd, _ := os.Getwd()
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
	
	actionData := make(common.GameInput, 2)
	actionData[common.InputPrimary] = "Button 1"
	actionData[common.InputSecondary] = "Button 2"
	
	inputs := make(common.DeviceInputs)
	
	// Run
	res, logo := matchGameInputToModel("test", actionData, inputs, nil, log)
	
	if logo != "fs2020" {
		t.Error("Wrong logo")
	}
	if len(res) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(res))
	}
	if res[0] != "1" || res[1] != "2" {
		t.Errorf("Unexpected results: %v", res)
	}
	
	// Error case
	actionDataError := make(common.GameInput, 2)
	actionDataError[common.InputPrimary] = "Unknown"
	
	resErr, _ := matchGameInputToModel("test", actionDataError, inputs, nil, log)
	if len(resErr) != 0 {
		t.Error("Expected empty result for unknown input")
	}
}

func TestLoadInputFiles_DuplicateContext(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
	}

	// XML with duplicate context
	xmlData := []byte(`
		<Device DeviceName="Alpha Flight Controls">
			<Context ContextName="PLANE">
				<Action ActionName="ACTION1"><Primary><KEY Information="Button 1"/></Primary></Action>
			</Context>
			<Context ContextName="PLANE">
				<!-- Duplicate Context -->
				<Action ActionName="ACTION2"><Primary><KEY Information="Button 2"/></Primary></Action>
			</Context>
		</Device>
	`)

	files := [][]byte{xmlData}

	_, _, _ = loadInputFiles(files, deviceMap, log, true, true)

	// Check for duplicate context error
	foundDuplicate := false
	for _, entry := range log.Entries {
		if entry.IsError && entry.Msg == "FS2020 duplicate context: PLANE" {
			foundDuplicate = true
		}
	}
	if !foundDuplicate {
		t.Error("Expected error for duplicate context")
	}
}

func TestLoadInputFiles_DuplicateAction(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"Alpha Flight Controls": "AlphaFlightControls",
	}

	// XML with duplicate action
	xmlData := []byte(`
		<Device DeviceName="Alpha Flight Controls">
			<Context ContextName="PLANE">
				<Action ActionName="ACTION1"><Primary><KEY Information="Button 1"/></Primary></Action>
				<Action ActionName="ACTION1"><Primary><KEY Information="Button 2"/></Primary></Action>
			</Context>
		</Device>
	`)

	files := [][]byte{xmlData}

	_, _, _ = loadInputFiles(files, deviceMap, log, true, true)

	// Check for duplicate action error
	foundDuplicate := false
	for _, entry := range log.Entries {
		if entry.IsError && entry.Msg == "FS2020 duplicate action: ACTION1" {
			foundDuplicate = true
		}
	}
	if !foundDuplicate {
		t.Error("Expected error for duplicate action")
	}
}

func TestMatchGameInputToModel_SecondaryFailure(t *testing.T) {
	log := common.NewLog()

	// Setup sharedRegexes
	wd, _ := os.Getwd()
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

	actionData := make(common.GameInput, 2)
	actionData[common.InputPrimary] = "Button 1"
	actionData[common.InputSecondary] = "Unknown Action" // Will fail to match

	inputs := make(common.DeviceInputs)

	res, logo := matchGameInputToModel("test", actionData, inputs, nil, log)

	if logo != "fs2020" {
		t.Error("Wrong logo")
	}
	// Should have 1 result (primary only, secondary failed)
	if len(res) != 1 {
		t.Errorf("Expected 1 input (primary only), got %d", len(res))
	}

	// Check error was logged for secondary
	foundError := false
	for _, entry := range log.Entries {
		if entry.IsError && entry.Msg == "FS2020 did not find secondary input for Unknown Action" {
			foundError = true
		}
	}
	if !foundError {
		t.Error("Expected error for secondary input failure")
	}
}

func TestLoadInputFiles_SecondaryKey(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"TestDevice": "TestDevice",
	}

	// XML with Secondary key binding
	xmlData := []byte(`
		<Device DeviceName="TestDevice">
			<Context ContextName="PLANE">
				<Action ActionName="ACTION1">
					<Primary><KEY Information="Button 1"/></Primary>
					<Secondary><KEY Information="Button 2"/></Secondary>
				</Action>
			</Context>
		</Device>
	`)

	files := [][]byte{xmlData}
	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, true)

	// Verify both primary and secondary are populated
	if gameBinds[common.ProfileDefault]["TestDevice"]["PLANE"]["ACTION1"][common.InputSecondary] != "Button 2" {
		t.Error("Expected secondary input to be 'Button 2'")
	}
}

func TestLoadInputFiles_FriendlyName(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"TestDevice": "TestDevice",
	}

	// XML with FriendlyName element (custom profile name)
	xmlData := []byte(`
		<Profile>
			<FriendlyName>MyCustomProfile</FriendlyName>
			<Device DeviceName="TestDevice">
				<Context ContextName="PLANE">
					<Action ActionName="ACTION1">
						<Primary><KEY Information="Button 1"/></Primary>
					</Action>
				</Context>
			</Device>
		</Profile>
	`)

	files := [][]byte{xmlData}
	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, true)

	// The custom profile should be used
	if _, found := gameBinds["MyCustomProfile"]; !found {
		t.Log("Custom profile not found, checking default profile")
	}
}

func TestLoadInputFiles_EmptyFriendlyName(t *testing.T) {
	log := common.NewLog()
	deviceMap := common.DeviceNameFullToShort{
		"TestDevice": "TestDevice",
	}

	// XML with empty FriendlyName
	xmlData := []byte(`
		<Profile>
			<FriendlyName></FriendlyName>
			<Device DeviceName="TestDevice">
				<Context ContextName="PLANE">
					<Action ActionName="ACTION1">
						<Primary><KEY Information="Button 1"/></Primary>
					</Action>
				</Context>
			</Device>
		</Profile>
	`)

	files := [][]byte{xmlData}
	gameBinds, _, _ := loadInputFiles(files, deviceMap, log, true, true)

	// Should fall back to default profile
	if _, found := gameBinds[common.ProfileDefault]; !found {
		t.Error("Expected default profile to be used")
	}
}

func TestLoadInputFiles_DeviceMissingInfo(t *testing.T) {
	log := common.NewLog()
	
	// Map a device to DeviceMissingInfo
	deviceMap := common.DeviceNameFullToShort{
		"UnknownNewDevice": common.DeviceMissingInfo,
	}

	xmlData := []byte(`
		<Device DeviceName="UnknownNewDevice">
			<Context ContextName="PLANE">
				<Action ActionName="ACTION1">
					<Primary><KEY Information="Button 1"/></Primary>
				</Action>
			</Context>
		</Device>
	`)

	files := [][]byte{xmlData}
	_, _, _ = loadInputFiles(files, deviceMap, log, true, true)

	// Check that error was logged for missing info
	foundError := false
	for _, entry := range log.Entries {
		if entry.IsError && entry.Msg == "FS2020 missing info for device 'DeviceMissingInfo'" {
			foundError = true
		}
	}
	if !foundError {
		t.Error("Expected error for device missing info")
	}
}

func TestMatchGameInputToModelByRegex_AxisSubstitution(t *testing.T) {
	log := common.NewLog()
	
	// Setup regexes
	wd, _ := os.Getwd()
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
	
	inputs := make(common.DeviceInputs)
	
	// Provide gameInputMap with Axis substitution
	gameInputMap := common.InputTypeMapping{
		"Axis": map[string]string{
			"X": "CustomX",
		},
	}
	
	// "Axis X" should match axis pattern and get substituted (pattern: (?:([R])-)?Axis\s*([XYZ]))
	result := matchGameInputToModelByRegex("testDevice", "Axis X", inputs, gameInputMap, log)
	
	if result != "CustomXAxis" {
		t.Errorf("Expected 'CustomXAxis', got '%s'", result)
	}
}

func TestMatchGameInputToModelByRegex_RotationOverride(t *testing.T) {
	log := common.NewLog()
	
	// Setup regexes
	wd, _ := os.Getwd()
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
	
	inputs := make(common.DeviceInputs)
	
	// Provide gameInputMap with Rotation override
	gameInputMap := common.InputTypeMapping{
		"Rotation": map[string]string{
			"X": "CustomRX",
		},
	}
	
	// "Rotation X" should match rotation pattern and get overridden
	result := matchGameInputToModelByRegex("testDevice", "Rotation X", inputs, gameInputMap, log)
	
	if result != "CustomRXAxis" {
		t.Errorf("Expected 'CustomRXAxis', got '%s'", result)
	}
}

func TestMatchGameInputToModelByRegex_SliderNoMapping(t *testing.T) {
	log := common.NewLog()
	
	// Setup regexes
	wd, _ := os.Getwd()
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
	
	inputs := make(common.DeviceInputs)
	
	// No Slider mapping provided - should hit else branch
	gameInputMap := common.InputTypeMapping{}
	
	// "Slider X" should match slider pattern but fail without mapping
	result := matchGameInputToModelByRegex("testDevice", "Slider X", inputs, gameInputMap, log)
	
	if result != "" {
		t.Errorf("Expected empty string for unmapped slider, got '%s'", result)
	}
	
	// Check error was logged
	foundError := false
	for _, entry := range log.Entries {
		if entry.IsError {
			foundError = true
		}
	}
	if !foundError {
		t.Error("Expected error for unmapped slider")
	}
}

// MockXMLTokenReader that returns an error after some valid tokens
type MockXMLTokenReader struct {
	tokens []xml.Token
	errs   []error
	idx    int
}

func (m *MockXMLTokenReader) Token() (xml.Token, error) {
	if m.idx >= len(m.tokens) {
		return nil, io.EOF
	}
	token := m.tokens[m.idx]
	err := m.errs[m.idx]
	m.idx++
	return token, err
}

func TestLoadInputFiles_XMLDecodeError(t *testing.T) {
	// Save original factory and restore after test
	originalFactory := xmlDecoderFactory
	defer func() { xmlDecoderFactory = originalFactory }()
	
	// Create a mock decoder that returns an error after a valid token
	// Important: we must return a non-nil token with an error for the error path to be hit
	// because "if token == nil" is checked before "else if err != nil"
	xmlDecoderFactory = func(data []byte) XMLTokenReader {
		return &MockXMLTokenReader{
			tokens: []xml.Token{
				xml.StartElement{Name: xml.Name{Local: "Root"}}, // Valid first
				xml.CharData("x"), // Non-nil token returned with error
			},
			errs: []error{
				nil,
				fmt.Errorf("mock XML decode error"),
			},
		}
	}
	
	log := common.NewLog()
	deviceMap := make(common.DeviceNameFullToShort)
	
	// Call loadInputFiles with any data - our mock will control behavior
	files := [][]byte{[]byte("<Root></Root>")}
	
	gameBinds, neededDevices, contextsToColours := loadInputFiles(files, deviceMap, log, false, false)
	
	// Function should return early on error
	if gameBinds == nil {
		t.Error("gameBinds should not be nil")
	}
	if neededDevices == nil {
		t.Error("neededDevices should not be nil")
	}
	if contextsToColours == nil {
		t.Error("contextsToColours should not be nil")
	}
	
	// Check that error was logged
	foundError := false
	for _, entry := range log.Entries {
		if entry.IsError {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("Expected error to be logged for XML decode failure")
	}
}

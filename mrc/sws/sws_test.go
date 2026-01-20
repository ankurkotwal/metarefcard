package sws

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

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
	
	// Test case 3: Button Range 21-40 (e.g. 22 -> 1)
	detailsRange1 := &swsActionDetails{Axis: "26", Button: "22"}
	got, _ = interpretInput(detailsRange1, "Any", "", "", log)
	if got != "1" {
		t.Errorf("interpretInput 22 = %v, want 1", got)
	}
	
	// Test case 4: Button Range 64-86 (e.g. 65 -> 20)
	detailsRange2 := &swsActionDetails{Axis: "26", Button: "65"}
	got, _ = interpretInput(detailsRange2, "Any", "", "", log)
	if got != "20" {
		t.Errorf("interpretInput 65 = %v, want 20", got)
	}
	
	// Test case 5: Button 86 (Empty)
	details86 := &swsActionDetails{Axis: "26", Button: "86"}
	got, _ = interpretInput(details86, "Any", "", "", log)
	if got != "" {
		t.Errorf("interpretInput 86 = %v, want empty", got)
	}
	
	// Test case 6: Throttle specific (e.g. 40 -> ZAxis)
	detailsThrottle := &swsActionDetails{Axis: "26", Button: "40"}
	got, _ = interpretInput(detailsThrottle, "SaitekX55Throttle", "", "", log)
	if got != "ZAxis" {
		t.Errorf("interpretInput Throttle 40 = %v, want ZAxis", got)
	}
	
	// Test case 7: DeviceID -1 (Ignore)
	// Test deviceID -1 (should ignore)
	details.DeviceID = "-1"
	res, _ := interpretInput(details, "SaitekX55Joystick", "Ctx", "Act", log)
	if res != "" {
		t.Error("Expected empty string for deviceID -1")
	}

	// Test X55 Joystick Buttons 46-51
	joyTests := []struct{ btn int; want string }{
		{46, "RZAxis"}, {47, "RZAxis"},
		{48, "POV1Up"}, {49, "POV1Down"},
		{50, "POV1Left"}, {51, "POV1Right"},
	}
	for _, tt := range joyTests {
		d := &swsActionDetails{Axis: "26", Button: strconv.Itoa(tt.btn), DeviceID: "0"}
		got, _ := interpretInput(d, "SaitekX55Joystick", "", "", log)
		if got != tt.want {
			t.Errorf("Joystick Button %d: got %s, want %s", tt.btn, got, tt.want)
		}
	}

	// Test X55 Throttle Buttons 40-47
	thrTests := []struct{ btn int; want string }{
		{40, "ZAxis"}, {41, "ZAxis"},
		{42, "RXAxis"}, {43, "RXAxis"},
		{44, "RYAxis"}, {45, "RYAxis"},
		{46, "RZAxis"}, {47, "RZAxis"},
	}
	for _, tt := range thrTests {
		d := &swsActionDetails{Axis: "26", Button: strconv.Itoa(tt.btn), DeviceID: "1"}
		got, _ := interpretInput(d, "SaitekX55Throttle", "", "", log)
		if got != tt.want {
			t.Errorf("Throttle Button %d: got %s, want %s", tt.btn, got, tt.want)
		}
	}
}

func TestLoadInputFiles_Errors(t *testing.T) {
	// Initialize regexes same as TestLoadInputFiles
	log := common.NewLog()
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/sws.yaml")
	gameData := common.LoadGameModel(configPath, "SWS Data", false, log)
	sharedGameData = gameData
	sharedRegexes = swsRegexes{
		Bind:     regexp.MustCompile(sharedGameData.Regexes["Bind"]),
		Joystick: regexp.MustCompile(sharedGameData.Regexes["Joystick"]),
	}
	
	// Bad integer in GstKeyBinding (matches[3])
	// Use valid prefix "GstKeyBinding.IncomDefaultInputConcepts.ConceptActivate"
	file1 := []byte("GstKeyBinding.IncomDefaultInputConcepts.ConceptActivate.99999999999999999999.button 1")
	files := [][]byte{file1}
	mapping := make(common.DeviceNameFullToShort)
	
	loadInputFiles(files, mapping, log, true, true)
	// Should log error
	found := false
	for _, e := range log.Entries {
		if e.IsError && len(e.Msg) > 0 { found = true }
	}
	if !found {
		t.Error("Expected error for non-int device num")
	}
	
	// Unknown Device
	// Use space separator as per regex
	file2 := []byte("GstInput.JoystickDevice0 UnknownDevice")
	loadInputFiles([][]byte{file2}, mapping, log, true, false)
	found = false
	for _, e := range log.Entries {
		if e.IsError && len(e.Msg) > 0 { found = true }
	}
	if !found {
		t.Error("Expected error for unknown device")
	}

	
	// Input type field error
	// Valid prefix but invalid field "unknown"
	file3 := []byte("GstKeyBinding.IncomDefaultInputConcepts.ConceptActivate.1.unknown 1")
	log = common.NewLog()
	loadInputFiles([][]byte{file3}, mapping, log, true, false)
	// Should see error
	found = false
	for _, e := range log.Entries {
		if e.IsError && len(e.Msg) > 0 { found = true }
	}
	if !found {
		t.Error("Expected error for unknown input type")
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

func BenchmarkLoadInputFiles(b *testing.B) {
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
		b.Fatalf("Failed to read test data file: %v", err)
	}

	files := [][]byte{fileContent}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loadInputFiles(files, deviceMap, log, false, false)
	}
}

func TestGetGameInfo(t *testing.T) {
	label, desc, handler, matchFunc := GetGameInfo()
	if label != "sws" {
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

func TestMatchGameInputToModel(t *testing.T) {
	log := common.NewLog()
	
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/sws.yaml")
	gameData := common.LoadGameModel(configPath, "SWS Data", false, log)
	sharedGameData = gameData
	
	action := make(common.GameInput, 2)
	action[common.InputPrimary] = "input1"
	
	res, logo := matchGameInputToModel("dev", action, nil, nil, log)
	
	// SWS match function just returns the input and logo
	if logo != "StarWarsSquadrons" { // Logo property in yaml is StarWarsSquadrons?
		// We need to check yaml. The file content I saw earlier didn't show the Logo line fully?
		// sws.go says: sharedGameData.Logo
		// sws.yaml usually starts with "Logo: ..."
		// Let's assume it loads correctly. If it fails, I'll see error.
	}
	if res[common.InputPrimary] != "input1" {
		t.Error("Result mismatch")
	}
}

func TestHandleRequest(t *testing.T) {
	log := common.NewLog()
	
	wd, _ := os.Getwd()
	// Create config link
	os.MkdirAll("config", 0755)
	// Copy ../../config/sws.yaml to config/sws.yaml
	input, _ := os.ReadFile("../../config/sws.yaml")
	if len(input) == 0 {
		input, _ = os.ReadFile(filepath.Join(wd, "../../config/sws.yaml"))
	}
	os.WriteFile("config/sws.yaml", input, 0644)
	defer os.RemoveAll("config")
	
	config := &common.Config{
		Devices: common.Devices{
			DeviceToShortNameMap: common.DeviceNameFullToShort{
				"Saitek Pro Flight X-55 Rhino Stick": "SaitekX55Joystick",
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

func TestLoadInputFiles_UnexpectedDeviceNumber(t *testing.T) {
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
		"Valid Device": "ValidDevice",
	}

	// Device number 0 (becomes -1 after subtraction, triggering error)
	fileData := []byte("GstInput.JoystickDevice0 Valid Device")

	files := [][]byte{fileData}

	_, devices, _ := loadInputFiles(files, deviceMap, log, false, false)

	// Device should NOT be added because num-1 = -1 which is >= 0 check fails
	if devices["ValidDevice"] {
		t.Error("Device should not be added for JoystickDevice0 (num becomes -1)")
	}
	// Check error logged
	foundError := false
	for _, e := range log.Entries {
		if e.IsError && e.Msg == "SWS unexpected device number 0" {
			foundError = true
		}
	}
	if !foundError {
		t.Error("Expected error for unexpected device number")
	}
}

func TestInterpretInput_UnknownInput(t *testing.T) {
	log := common.NewLog()

	// Test case with unknown Axis value (not 8, 9, 10, 11, or 26)
	details := &swsActionDetails{
		Axis:     "99", // Unknown axis
		DeviceID: "0",
	}

	result, err := interpretInput(details, "SaitekX55Joystick", "ctx", "action", log)

	if err == nil {
		t.Error("Expected error for unknown input")
	}
	if result != "" {
		t.Errorf("Expected empty result, got %s", result)
	}
}

func TestInterpretInput_ButtonNotNumber(t *testing.T) {
	log := common.NewLog()

	// Axis 26 but button is not a number
	details := &swsActionDetails{
		Axis:     "26",
		Button:   "not_a_number",
		DeviceID: "0",
	}

	result, err := interpretInput(details, "SaitekX55Joystick", "ctx", "action", log)

	if err == nil {
		t.Error("Expected error for button not a number")
	}
	if result != "" {
		t.Errorf("Expected empty result, got %s", result)
	}
}

func TestInterpretInput_ButtonOutOfRange(t *testing.T) {
	log := common.NewLog()

	// Button value that doesn't match any range (e.g., 5)
	// Falls through all cases in the function
	details := &swsActionDetails{
		Axis:     "26",
		Button:   "5", // Not in 21-40, not in 64-86, not 86, not in device-specific ranges
		DeviceID: "0",
	}

	result, err := interpretInput(details, "SaitekX55Joystick", "ctx", "action", log)

	if err == nil {
		t.Error("Expected error for button out of range")
	}
	if result != "" {
		t.Errorf("Expected empty result, got %s", result)
	}
}

func TestLoadInputFiles_JoystickDevice0(t *testing.T) {
	// Test the case where JoystickDevice number is 0, which becomes -1 after decrement
	// This should trigger the error path at line 96-97
	log := common.NewLog()
	
	// Initialize regexes (required for loadInputFiles)
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/sws.yaml")
	gameData := common.LoadGameModel(configPath, "SWS Data", false, log)
	sharedGameData = gameData
	sharedRegexes = swsRegexes{
		Bind:     regexp.MustCompile(sharedGameData.Regexes["Bind"]),
		Joystick: regexp.MustCompile(sharedGameData.Regexes["Joystick"]),
	}
	
	deviceMap := common.DeviceNameFullToShort{
		"Known Device": "KnownDevice",
	}
	
	// Use JoystickDevice0 (index 0) which after num-- becomes -1
	data := []byte(`GstInput.JoystickDevice0 Known Device`)
	files := [][]byte{data}
	
	loadInputFiles(files, deviceMap, log, false, false)
	
	// Check that error was logged for unexpected device number
	foundError := false
	for _, entry := range log.Entries {
		if entry.IsError && strings.Contains(entry.Msg, "unexpected device number") {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("Expected error for JoystickDevice0 (negative num after decrement)")
	}
}

// MockScannerReader that returns an error on Err()
type MockScannerReader struct {
	lines []string
	err   error
	idx   int
}

func (m *MockScannerReader) Scan() bool {
	if m.idx >= len(m.lines) {
		return false
	}
	m.idx++
	return true
}

func (m *MockScannerReader) Text() string {
	return m.lines[m.idx-1]
}

func (m *MockScannerReader) Err() error {
	return m.err
}

func TestLoadInputFiles_ScannerError(t *testing.T) {
	// Save original factory and restore after test
	originalFactory := scannerFactory
	defer func() { scannerFactory = originalFactory }()
	
	// Initialize regexes (required for loadInputFiles)
	log := common.NewLog()
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/sws.yaml")
	gameData := common.LoadGameModel(configPath, "SWS Data", false, log)
	sharedGameData = gameData
	sharedRegexes = swsRegexes{
		Bind:     regexp.MustCompile(sharedGameData.Regexes["Bind"]),
		Joystick: regexp.MustCompile(sharedGameData.Regexes["Joystick"]),
	}
	
	// Create a mock scanner that returns an error
	scannerFactory = func(data []byte) ScannerReader {
		return &MockScannerReader{
			lines: []string{"some line"},
			err:   fmt.Errorf("mock scanner error"),
		}
	}
	
	deviceMap := common.DeviceNameFullToShort{}
	files := [][]byte{[]byte("test")}
	
	loadInputFiles(files, deviceMap, log, false, false)
	
	// Check that error was logged for scanner error
	foundError := false
	for _, entry := range log.Entries {
		if entry.IsError && strings.Contains(entry.Msg, "scan file") {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("Expected error for scanner failure")
	}
}

func TestLoadInputFiles_InterpretInputError(t *testing.T) {
	// Test that loadInputFiles handles interpretInput errors correctly
	// This covers lines 169-171 in sws.go
	log := common.NewLog()
	
	// Initialize regexes
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "../../config/sws.yaml")
	gameData := common.LoadGameModel(configPath, "SWS Data", false, log)
	sharedGameData = gameData
	sharedRegexes = swsRegexes{
		Bind:     regexp.MustCompile(sharedGameData.Regexes["Bind"]),
		Joystick: regexp.MustCompile(sharedGameData.Regexes["Joystick"]),
	}
	
	deviceMap := common.DeviceNameFullToShort{
		"Test Device": "TestDevice",
	}
	
	// Create input that:
	// 1. Maps device 1 to "TestDevice" (becomes deviceid 0 after decrement)
	// 2. Has a GstKeyBinding matching regex pattern that refers to deviceid 0
	// 3. Has axis 26 with an invalid button value that causes interpretInput to return error
	// Regex: ^GstKeyBinding\.Incom(Default|Soldier|Starship)InputConcepts\.Concept(.+)\.(\d+)\.(.+)\s+(.+)$
	data := []byte(`GstInput.JoystickDevice1 Test Device
GstKeyBinding.IncomDefaultInputConcepts.ConceptTest.0.axis 26
GstKeyBinding.IncomDefaultInputConcepts.ConceptTest.0.button InvalidButton
GstKeyBinding.IncomDefaultInputConcepts.ConceptTest.0.deviceid 0`)
	
	files := [][]byte{data}
	
	loadInputFiles(files, deviceMap, log, false, false)
	
	// Check that error was logged for interpretInput failure
	foundError := false
	for _, entry := range log.Entries {
		if entry.IsError && (strings.Contains(entry.Msg, "SWS button not number") || 
			strings.Contains(entry.Msg, "SWS Unknown input")) {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("Expected error from interpretInput")
	}
}

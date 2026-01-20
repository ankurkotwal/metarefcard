package common

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadGameModel(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "model.yaml")
	data := GameData{
		Logo: "logo.jpg",
	}
	bytes, _ := yaml.Marshal(data)
	os.WriteFile(path, bytes, 0644)

	log, _ := mockLogger()
	result := LoadGameModel(path, "label", false, log)
	if result.Logo != "logo.jpg" {
		t.Errorf("LoadGameModel failed to load logo")
	}
}

func TestFilterDevices(t *testing.T) {
	// Setup
	config := &Config{
		Devices: Devices{
			Index: DeviceMap{
				"d1": DeviceInputs{"i1": InputData{}},
				"d2": DeviceInputs{"i2": InputData{}},
			},
		},
		DebugOutput: true,
	}
	log, _ := mockLogger()
	
	// Case 1: Select both (d2 explicitly, d1 via neededDevices logic inside calling code usually - wait, logic is:
	// "Filter for only the device groups we're interested in"
	// loop over keys of neededDevices. check if in config.
	
	needed := make(Set)
	needed["d1"] = true
	needed["d3"] = true // Missing one
	
	filtered := FilterDevices(needed, config, log)
	
	if _, ok := filtered["d1"]; !ok {
		t.Error("d1 should be present")
	}
	if _, ok := filtered["d2"]; ok {
		t.Error("d2 should not be present (not in needed)")
	}
	if _, ok := filtered["d3"]; ok {
		t.Error("d3 should not be present (not in config)")
	}
}

func TestGenerateContextColours(t *testing.T) {
	config := &Config{
		AlternateColours: []string{"c1", "c2"},
	}
	contexts := make(ContextToColours)
	contexts["A"] = ""
	contexts["B"] = ""
	contexts["C"] = ""
	
	GenerateContextColours(contexts, config)
	
	// keys sorted: A, B, C
	// A -> c1 (i=0 -> 1)
	// B -> c2 (i=1 -> 2)
	// C -> c1 (i=2 -> 0 -> 1)
	
	if contexts["A"] != "c1" || contexts["B"] != "c2" || contexts["C"] != "c1" {
		t.Errorf("Unexpected colours: %v", contexts)
	}
}

func TestPopulateImageOverlays(t *testing.T) {
	// Complex Setup
	log, _ := mockLogger()
	
	// Config
	config := &Config{
		Devices: Devices{
			Index: DeviceMap{
				"d1": DeviceInputs{
					"btn1": InputData{X: 10, Y: 10},
					"btn2": InputData{X: 0, Y: 0}, // Invalid location
				},
			},
			ImageMap: ImageMap{
				"d1": "d1.jpg",
			},
		},
	}
	
	needed := make(Set)
	needed["d1"] = true
	
	// Game Binds
	binds := make(GameBindsByProfile)
	profile := "Default"
	binds[profile] = make(GameDeviceContextActions)
	
	// Context
	binds[profile]["d1"] = make(GameContextActions)
	binds[profile]["d1"]["ctx1"] = make(GameActions)
	
	// Actions
	// 1. Success
	binds[profile]["d1"]["ctx1"]["action1"] = []string{"input1", ""}
	// 2. Button 2 (invalid loc)
	binds[profile]["d1"]["ctx1"]["action2"] = []string{"input2", ""}
	// 3. Unknown button (via matchFunc)
	binds[profile]["d1"]["ctx1"]["action3"] = []string{"input3", ""}

	gameData := GameData{
		InputLabels: map[string]string{
			"action1": "Start",
		},
		InputMap: make(DeviceInputTypeMapping),
	}
	
	// MatchFunc
	matchFunc := func(deviceName string, actionData GameInput,
		deviceInputs DeviceInputs, gameInputMap InputTypeMapping, log *Logger) (GameInput, string) {
		
		input := actionData[InputPrimary]
		if input == "input1" {
			return []string{"btn1"}, "FoundBtn1"
		}
		if input == "input2" {
			return []string{"btn2"}, "FoundBtn2"
		}
		if input == "input3" {
			return []string{"btn3"}, "FoundBtn3" // btn3 implies missing in deviceInputs
		}
		return []string{}, ""
	}
	
	overlays := PopulateImageOverlays(needed, config, log, binds, gameData, matchFunc)
	
	// Verify
	imgOverlays := overlays[profile]["d1.jpg"]
	if imgOverlays == nil {
		t.Fatal("No overlays for d1.jpg")
	}
	
	// Check action1 -> btn1
	key := "d1:btn1"
	od, ok := imgOverlays[key]
	if !ok {
		t.Error("Overlay for btn1 missing")
	}
	if len(od.ContextToTexts["ctx1"]) == 0 || od.ContextToTexts["ctx1"][0] != "Start" {
		t.Errorf("Incorrect text for btn1: %v", od.ContextToTexts)
	}
	
	// Check action2 -> btn2 (should be skipped/logged error due to 0,0 location)
	key2 := "d1:btn2"
	if _, ok := imgOverlays[key2]; ok {
		t.Error("Overlay for btn2 should be missing (location 0,0)")
	}
	
	// Check action3 -> btn3 (should be skipped/logged error due to missing input)
	key3 := "d1:btn3"
	if _, ok := imgOverlays[key3]; ok {
		t.Error("Overlay for btn3 should be missing (unknown input)")
	}
	
	// Test Concatenation (Secondary overlay on same button)
	// We need to simulate adding another action to btn1
	// Let's call GenerateImageOverlays directly for this or modify mocks. 
	// Easier to add to binds above but iterating map order is undefined.
	
	// Let's check GenerateImageOverlays directly for concat logic
	
	// Test GenerateImageOverlays directly for label fallback
	log2, _ := mockLogger()
	existing := make(OverlaysByImage)
	inputData := InputData{X: 10, Y: 10}
	
	// First call
	GenerateImageOverlays(existing, "btn_x", inputData, gameData, "UnknownAction", "ctxA", "devX", "imgX.jpg", "label", log2)
	
	// Should use "UnknownAction" as text since not in InputLabels
	odX := existing["imgX.jpg"]["devX:btn_x"]
	if odX.ContextToTexts["ctxA"][0] != "UnknownAction" {
		t.Errorf("Expected UnknownAction, got %s", odX.ContextToTexts["ctxA"][0])
	}
	
	// Second call - append
	GenerateImageOverlays(existing, "btn_x", inputData, gameData, "action1", "ctxA", "devX", "imgX.jpg", "label", log2)
	// action1 maps to "Start" in gameData above
	
	texts := existing["imgX.jpg"]["devX:btn_x"].ContextToTexts["ctxA"]
	if len(texts) != 2 {
		t.Errorf("Expected 2 texts, got %d", len(texts))
	}
	// "Start", "UnknownAction" sorted -> Start, UnknownAction? S < U.
	sort.Strings(texts)
	if texts[0] != "Start" || texts[1] != "UnknownAction" {
		t.Errorf("Unexpected text order: %v", texts)
	}
}

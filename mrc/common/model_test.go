package common

import (
	"os"
	"testing"
)

func TestFilterDevices_Unknown(t *testing.T) {
	log := NewLog()
	config := &Config{
		Devices: Devices{
			Index: map[string]DeviceInputs{
				"KnownDevice": {},
			},
		},
		DebugOutput: true,
	}

	neededDevices := Set{
		"KnownDevice":   true,
		"UnknownDevice": true,
	}

	filtered := FilterDevices(neededDevices, config, log)

	if _, found := filtered["KnownDevice"]; !found {
		t.Error("Expected KnownDevice to be present")
	}
	if _, found := filtered["UnknownDevice"]; found {
		t.Error("Expected UnknownDevice to be filtered out")
	}
}

func TestLoadGameModel_Error(t *testing.T) {
	log := NewLog()
	_, err := LoadGameModel("missing_model.yaml", "Test", true, log)
	if err == nil {
		t.Error("Expected error for missing model file")
	}
}

func TestLoadGameModel_Success(t *testing.T) {
	// Create a temporary game model file
	tmpFile, err := os.CreateTemp("", "game_model_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `
Logo: "test_logo"
Regexes:
  Button: "Button ([0-9]+)"
  Axis: "Axis ([X,Y,Z])"
InputMap:
  TestDevice:
    Axis:
      X: "U"
InputLabels:
  ACTION_TEST: "Test Action"
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	log := NewLog()
	gameData, err := LoadGameModel(tmpFile.Name(), "TestGame", false, log)

	if err != nil {
		t.Fatalf("LoadGameModel failed: %v", err)
	}
	if gameData.Logo != "test_logo" {
		t.Errorf("Expected Logo 'test_logo', got '%s'", gameData.Logo)
	}
	if gameData.Regexes["Button"] != "Button ([0-9]+)" {
		t.Error("Expected Button regex")
	}
	if gameData.InputLabels["ACTION_TEST"] != "Test Action" {
		t.Error("Expected InputLabels")
	}
}

func TestGenerateContextColours(t *testing.T) {
	contexts := ContextToColours{
		"Context1": "",
		"Context2": "",
		"Context3": "",
	}

	config := &Config{
		AlternateColours: []string{"#FF0000", "#00FF00", "#0000FF"},
	}

	GenerateContextColours(contexts, config)

	// Verify all contexts have been assigned colours
	for context, colour := range contexts {
		if colour == "" {
			t.Errorf("Context %s was not assigned a colour", context)
		}
	}

	// Verify colors are from the palette
	validColours := map[string]bool{
		"#FF0000": true,
		"#00FF00": true,
		"#0000FF": true,
	}
	for context, colour := range contexts {
		if !validColours[colour] {
			t.Errorf("Context %s has invalid colour %s", context, colour)
		}
	}
}

func TestGenerateContextColours_Wraparound(t *testing.T) {
	// More contexts than colours - should wrap around
	contexts := ContextToColours{
		"A": "",
		"B": "",
		"C": "",
		"D": "",
		"E": "",
	}

	config := &Config{
		AlternateColours: []string{"#111", "#222"},
	}

	GenerateContextColours(contexts, config)

	// All should be assigned
	for context, colour := range contexts {
		if colour == "" {
			t.Errorf("Context %s was not assigned a colour", context)
		}
	}
}

func TestGenerateImageOverlays_NewImage(t *testing.T) {
	overlaysByImage := make(OverlaysByImage)
	inputData := InputData{X: 10, Y: 20, W: 100, H: 50}
	gameData := GameData{
		InputLabels: map[string]string{
			"ACTION_1": "Action One",
		},
	}
	log := NewLog()

	GenerateImageOverlays(overlaysByImage, "Button1", inputData, gameData,
		"ACTION_1", "TestContext", "TestDevice", "test_image", "TestGame", log)

	// Verify image was added
	if _, found := overlaysByImage["test_image"]; !found {
		t.Error("Expected test_image in overlaysByImage")
	}

	// Verify overlay data
	deviceInput := "TestDevice:Button1"
	if overlay, found := overlaysByImage["test_image"][deviceInput]; found {
		if texts, ok := overlay.ContextToTexts["TestContext"]; ok {
			if len(texts) != 1 || texts[0] != "Action One" {
				t.Errorf("Expected ['Action One'], got %v", texts)
			}
		} else {
			t.Error("Expected TestContext in ContextToTexts")
		}
	} else {
		t.Errorf("Expected %s in overlay", deviceInput)
	}
}

func TestGenerateImageOverlays_ExistingImageNewInput(t *testing.T) {
	overlaysByImage := make(OverlaysByImage)
	overlaysByImage["existing_image"] = make(map[string]OverlayData)

	inputData := InputData{X: 30, Y: 40, W: 80, H: 60}
	gameData := GameData{
		InputLabels: map[string]string{},
	}
	log := NewLog()

	GenerateImageOverlays(overlaysByImage, "Button2", inputData, gameData,
		"ACTION_2", "Context2", "Device2", "existing_image", "Game", log)

	deviceInput := "Device2:Button2"
	if overlay, found := overlaysByImage["existing_image"][deviceInput]; found {
		// Since no InputLabel, actionName should be used as text
		if texts := overlay.ContextToTexts["Context2"]; len(texts) != 1 || texts[0] != "ACTION_2" {
			t.Errorf("Expected ['ACTION_2'], got %v", texts)
		}
	} else {
		t.Error("Expected overlay for Device2:Button2")
	}
}

func TestGenerateImageOverlays_ExistingOverlayConcatenation(t *testing.T) {
	overlaysByImage := make(OverlaysByImage)
	
	// Pre-populate with existing overlay
	existingOverlay := OverlayData{
		ContextToTexts: map[string][]string{
			"SharedContext": {"First Action"},
		},
		PosAndSize: InputData{X: 10, Y: 10, W: 50, H: 30},
	}
	overlaysByImage["shared_image"] = map[string]OverlayData{
		"SharedDevice:SharedButton": existingOverlay,
	}

	inputData := InputData{X: 10, Y: 10, W: 50, H: 30}
	gameData := GameData{
		InputLabels: map[string]string{
			"ACTION_SECOND": "Second Action",
		},
	}
	log := NewLog()

	// Add another action to the same device:input
	GenerateImageOverlays(overlaysByImage, "SharedButton", inputData, gameData,
		"ACTION_SECOND", "SharedContext", "SharedDevice", "shared_image", "Game", log)

	deviceInput := "SharedDevice:SharedButton"
	if overlay, found := overlaysByImage["shared_image"][deviceInput]; found {
		texts := overlay.ContextToTexts["SharedContext"]
		if len(texts) != 2 {
			t.Errorf("Expected 2 texts after concatenation, got %d", len(texts))
		}
		// Texts should be sorted
		if texts[0] != "First Action" || texts[1] != "Second Action" {
			t.Errorf("Expected sorted texts ['First Action', 'Second Action'], got %v", texts)
		}
	} else {
		t.Error("Expected overlay for SharedDevice:SharedButton")
	}
}


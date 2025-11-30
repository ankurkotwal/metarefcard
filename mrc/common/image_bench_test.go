package common

import (
	"path/filepath"
	"testing"
)

// Mock types to help with testing
func TestPopulateImageOverlays(t *testing.T) {
	// Setup
	log := NewLog()
	config := &Config{}
	// Load a minimal config
	config.Devices.Index = make(map[string]DeviceInputs)
	config.Devices.ImageMap = make(map[string]string)

	deviceName := "test_device"
	inputName := "trigger"
	imageName := "test_image.jpg"

	config.Devices.Index[deviceName] = map[string]InputData{
		inputName: {X: 10, Y: 10, W: 100, H: 100},
	}
	config.Devices.ImageMap[deviceName] = imageName

	neededDevices := make(Set)
	neededDevices[deviceName] = true

	gameBinds := make(GameBindsByProfile)
	gameBinds[ProfileDefault] = make(GameDeviceContextActions)
	gameBinds[ProfileDefault][deviceName] = make(GameContextActions)
	context := "Flight"
	action := "Fire"
	gameBinds[ProfileDefault][deviceName][context] = make(GameActions)
	gameBinds[ProfileDefault][deviceName][context][action] = []string{inputName, ""}

	gameData := GameData{
		InputLabels: map[string]string{
			action: "Fire Weapons",
		},
		InputMap: make(DeviceInputTypeMapping),
	}

	matchFunc := func(deviceName string, actionData GameInput,
		deviceInputs DeviceInputs, gameInputMap InputTypeMapping, log *Logger) (GameInput, string) {
		// Mock matcher always returns the input name
		return []string{inputName, ""}, "MockLabel"
	}

	// Test
	overlays := PopulateImageOverlays(neededDevices, config, log, gameBinds, gameData, matchFunc)

	// Verify
	profileOverlays, ok := overlays[ProfileDefault]
	if !ok {
		t.Fatal("Expected default profile")
	}

	imageOverlays, ok := profileOverlays[imageName]
	if !ok {
		t.Fatal("Expected test image")
	}

	key := deviceName + ":" + inputName
	overlayData, ok := imageOverlays[key]
	if !ok {
		t.Fatal("Expected overlay data for key " + key)
	}

	texts, ok := overlayData.ContextToTexts[context]
	if !ok {
		t.Fatal("Expected context")
	}

	if len(texts) != 1 || texts[0] != "Fire Weapons" {
		t.Errorf("Expected 'Fire Weapons', got %v", texts)
	}
}

func BenchmarkGenerateImages(b *testing.B) {
	// Setup config with paths to real resources
	config := &Config{
		DefaultImage:     Dimensions2d{W: 1000, H: 1000},
		PixelMultiplier:  1.0,
		HotasImagesDir:   "../../resources/hotas-images",
		LogoImagesDir:    "../../resources/game-logos",
		FontsDir:         "../../resources/fonts",
		InputFont:        "YanoneKaffeesatz-Regular.ttf",
		InputFontSize:    20,
		InputMinFontSize: 10,
		JpgQuality:       80,
		ImageHeader: HeaderData{
			Font:     "Orbitron-Regular.ttf",
			FontSize: 20,
			Inset:    Point2d{X: 10, Y: 10},
		},
		Watermark: WatermarkData{
			Font:     "Dirga.ttf",
			FontSize: 10,
			Location: Point2d{X: 10, Y: 10},
		},
		Devices: Devices{
			DeviceLabelsByImage: map[string]string{
				"alphaflight": "Alpha Flight Controls",
			},
			ImageSizeOverride: map[string]Dimensions2d{},
		},
		BackgroundColour: "#000000",
		LightColour:      "#FFFFFF",
		DarkColour:       "#333333",
	}

	log := NewLog()

	// Ensure paths are absolute
	absHotasDir, _ := filepath.Abs(config.HotasImagesDir)
	config.HotasImagesDir = absHotasDir

	absLogoDir, _ := filepath.Abs(config.LogoImagesDir)
	config.LogoImagesDir = absLogoDir

	absFontsDir, _ := filepath.Abs(config.FontsDir)
	config.FontsDir = absFontsDir

	// Prepare overlays
	overlays := make(OverlaysByProfile)
	imageName := "alphaflight" // Use a real image
	overlays[ProfileDefault] = make(OverlaysByImage)
	overlays[ProfileDefault][imageName] = make(map[string]OverlayData)

	// Add dummy overlay data
	overlayData := OverlayData{
		PosAndSize: InputData{X: 100, Y: 100, W: 200, H: 50},
		ContextToTexts: map[string][]string{
			"Default": {"Test Label"},
		},
	}
	overlays[ProfileDefault][imageName]["test_device:trigger"] = overlayData

	categories := map[string]string{
		"Default": "#FF0000",
	}
	gameLabel := "fs2020" // Use a real logo if possible (fs2020.jpg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateImages(overlays, categories, gameLabel, config, log)
	}
}

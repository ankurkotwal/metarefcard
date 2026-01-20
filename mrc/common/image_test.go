package common

import (
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Mock FontLoader
type MockFontLoader struct {
	Face font.Face
}

func (m *MockFontLoader) LoadFont(dir string, name string, size int) font.Face {
	return m.Face
}

func TestGetPixelMultiplier(t *testing.T) {
	config := &Config{
		PixelMultiplier: 1.5,
		DefaultImage: Dimensions2d{W: 100, H: 100},
		Devices: Devices{
			ImageSizeOverride: map[string]Dimensions2d{
				"override": {W: 200, H: 200},
			},
		},
	}
	
	// Case 1: Default
	m1 := getPixelMultiplier("normal", config)
	if m1 != 1.5 {
		t.Errorf("Expected 1.5, got %f", m1)
	}
	
	// Case 2: Override
	m2 := getPixelMultiplier("override", config)
	if m2 != 2.0 {
		t.Errorf("Expected 2.0 (200/100), got %f", m2)
	}
}

func TestPrepareContexts(t *testing.T) {
	c := map[string][]string{
		"B": {},
		"A": {},
	}
	res := prepareContexts(c)
	if res[0] != "A" || res[1] != "B" {
		t.Errorf("Contexts not sorted: %v", res)
	}
}

func TestCalcFontSize(t *testing.T) {
	// We need a real font or a mock that behaves predictably.
	// Since we can't easily make a truetype font that measures exactly as we want without complex setup,
	// let's retry using real fonts if possible, or assume LoadFont works (tested in util).
	
	// The problem is `measureString` uses the font face.
	// If we use the mock loader, we can return a mock face?
	// `font.Face` is an interface.
	
	// Let's rely on the real font loading which we verified in `util_test.go`
	fontDir := "../../resources/fonts"
	fontName := "Dirga.ttf"
	
	// Only run if font exists (it should)
	// calcFontSize logic:
	// it tries maxFontSize (targetHeight). measure.
	// if y > targetHeight -> panic
	// if x > targetWidth -> reduce size (binary search?)
	// else -> increase size
	
	// We test simple converge.
	targetW := 100
	targetH := 20
	minSize := 5
	
	// Need a FontLoader that returns real fonts
	loader := NewFontFaceCache()
	
	size := calcFontSize("test", loader, targetH, targetW, targetH, fontDir, fontName, minSize)
	
	if size > targetH {
		t.Errorf("Size %d > Target Height %d", size, targetH)
	}
	if size < minSize {
		t.Errorf("Size %d < Min Size %d", size, minSize)
	}
}

func TestCalcFontSize_Panic(t *testing.T) {
	// Test that calcFontSize panics when y > targetHeight
	// We use a mock font loader that returns a mock font face with oversized metrics
	
	mockLoader := &MockOversizedFontLoader{}
	
	targetW := 1000
	targetH := 10 // Small target height
	minSize := 5
	
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when text height exceeds target height")
		}
	}()
	
	// This should panic because the mock font returns height > targetHeight
	calcFontSize("Test", mockLoader, targetH, targetW, targetH, "", "", minSize)
}

// MockOversizedFontLoader returns a font face that reports oversized height
type MockOversizedFontLoader struct{}

func (m *MockOversizedFontLoader) LoadFont(dir string, name string, size int) font.Face {
	return &MockOversizedFontFace{size: size}
}

// MockOversizedFontFace implements font.Face with oversized metrics
type MockOversizedFontFace struct {
	size int
}

func (f *MockOversizedFontFace) Close() error { return nil }

func (f *MockOversizedFontFace) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	return image.Rectangle{}, nil, image.Point{}, 0, false
}

func (f *MockOversizedFontFace) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	return fixed.Rectangle26_6{}, 0, false
}

func (f *MockOversizedFontFace) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	return fixed.I(10), true
}

func (f *MockOversizedFontFace) Kern(r0, r1 rune) fixed.Int26_6 {
	return 0
}

func (f *MockOversizedFontFace) Metrics() font.Metrics {
	// Return height that is much larger than the requested size
	// This will cause y > targetHeight and trigger the panic
	return font.Metrics{
		Height: fixed.I(f.size * 10), // 10x the expected height
	}
}

// Test for the edge case where delta==0 when trying to grow
func TestCalcFontSize_DeltaZeroEdgeCase(t *testing.T) {
	// Create a mock that behaves in a specific way to trigger the delta==0 branch
	// in the growth path (line 273)
	
	// To hit this, we need a scenario where after some iterations:
	// - x <= targetWidth (text fits)
	// - newFontSize != maxFontSize && newFontSize != maxFontSize-1
	// - delta = (maxFontSize - newFontSize) / 2 == 0
	
	// This is hard with real fonts, but we can use a mock that returns
	// specific widths based on the font size
	mockLoader := &MockEdgeCaseFontLoader{}
	
	// Start with targetHeight = 10
	// Initially: maxFontSize = 10, newFontSize = 10
	// We need to manipulate widths so the algorithm bounces around
	targetW := 50
	targetH := 10
	minSize := 1
	
	// The mock will return:
	// - For size 10: width = 100 (too big, reduce)
	// - For size 5: width = 30 (fits, try to grow)
	// - After bouncing, should hit delta == 0
	
	size := calcFontSize("Test", mockLoader, targetH, targetW, targetH, "", "", minSize)
	
	// Just verify it returns without panic and returns a valid size
	if size < minSize || size > targetH {
		t.Errorf("Size %d out of expected range [%d, %d]", size, minSize, targetH)
	}
}

// MockEdgeCaseFontLoader returns fonts with controlled width behavior
type MockEdgeCaseFontLoader struct{}

func (m *MockEdgeCaseFontLoader) LoadFont(dir string, name string, size int) font.Face {
	return &MockEdgeCaseFontFace{size: size}
}

type MockEdgeCaseFontFace struct {
	size int
}

func (f *MockEdgeCaseFontFace) Close() error { return nil }

func (f *MockEdgeCaseFontFace) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	return image.Rectangle{}, nil, image.Point{}, 0, false
}

func (f *MockEdgeCaseFontFace) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	return fixed.Rectangle26_6{}, 0, false
}

func (f *MockEdgeCaseFontFace) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	// Return width based on font size to control the algorithm
	// Large sizes return large advances, small sizes return small advances
	return fixed.I(f.size * 3), true // width per char = size * 3
}

func (f *MockEdgeCaseFontFace) Kern(r0, r1 rune) fixed.Int26_6 {
	return 0
}

func (f *MockEdgeCaseFontFace) Metrics() font.Metrics {
	// Return height proportional to size (but smaller than size to avoid panic)
	return font.Metrics{
		Height: fixed.I(f.size), // height = size
	}
}

func TestPopulateImage(t *testing.T) {
	// Integration test for populateImage
	// Setup Context
	dc := gg.NewContext(100, 100)
	
	// Setup Data
	overlays := make(map[string]OverlayData)
	overlays["test"] = OverlayData{
		PosAndSize: InputData{X: 10, Y: 10, W: 50, H: 20},
		ContextToTexts: map[string][]string{
			"ctx": {"Hello"},
		},
	}
	
	categories := map[string]string{"ctx": "#FFFFFF"}
	config := &Config{
		InputPixelXInset: 1,
		InputPixelYInset: 1,
		JpgQuality: 80,
		FontsDir: "../../resources/fonts",
		InputFont: "Dirga.ttf",
		InputMinFontSize: 5,
		LightColour: "#000000",
	}
	log, _ := mockLogger()
	loader := NewFontFaceCache()
	
	// Test
	buf := populateImage(dc, "img.jpg", image.Point{X: 100, Y: 100}, 1.0, overlays, categories, config, log, loader)
	
	if buf.Len() == 0 {
		t.Error("Buffer empty")
	}
	
	// Test Error Path: Overlay outside bounds
	overlays["bad"] = OverlayData{
		PosAndSize: InputData{X: 200, Y: 200, W: 10, H: 10},
	}
	log2, _ := mockLogger() // wait, populateImage calls log.Err not fatal.
	// log.Err does not modify *fatal bool.
	
	populateImage(dc, "img.jpg", image.Point{X: 100, Y: 100}, 1.0, overlays, categories, config, log2, loader)
	
	// Check log entries
	if len(log2.Entries) == 0 {
		t.Error("Should have logged error for out of bounds")
	}
}

func TestDecodeJpg_Error(t *testing.T) {
	log, _ := mockLogger()
	_, err := decodeJpg("nonexistent.jpg", log)
	if err == nil {
		t.Error("Should fail on missing file")
	}
	if len(log.Entries) == 0 {
		t.Error("Should log error")
	}
}

func TestGenerateImages(t *testing.T) {
	// Setup folders
	tmpDir := t.TempDir()
	fontsDir := "../../resources/fonts"
	logoDir := filepath.Join(tmpDir, "logo")
	hotasDir := filepath.Join(tmpDir, "hotas")
	os.Mkdir(logoDir, 0755)
	os.Mkdir(hotasDir, 0755)
	
	// Create dummy JPGs
	createDummyJpg(t, filepath.Join(logoDir, "game.jpg"))
	createDummyJpg(t, filepath.Join(hotasDir, "dev.jpg"))
	
	// Config
	config := &Config{
		LogoImagesDir: logoDir,
		HotasImagesDir: hotasDir,
		FontsDir: fontsDir,
		InputFont: "Dirga.ttf",
		InputFontSize: 12,
		InputMinFontSize: 5,
		ImageHeader: HeaderData{
			Font: "Dirga.ttf",
			FontSize: 14,
		},
		Watermark: WatermarkData{
			Font: "Dirga.ttf",
			FontSize: 10,
		},
		DefaultImage: Dimensions2d{W: 100, H: 100},
		PixelMultiplier: 1.0,
		Devices: Devices{
			DeviceLabelsByImage: map[string]string{
				"dev": "Device Label",
			},
			ImageSizeOverride: make(map[string]Dimensions2d),
		},
	}
	
	// Overlays
	overlays := make(OverlaysByProfile)
	overlays["Default"] = make(OverlaysByImage)
	overlays["Default"]["dev"] = map[string]OverlayData{
		"k1": {PosAndSize: InputData{X: 10, Y: 10, W: 20, H: 20}, ContextToTexts: map[string][]string{"c": {"T"}}},
	}
	
	categories := map[string]string{"c": "#FFFFFF"}
	log, _ := mockLogger()
	
	// Run
	files, size := GenerateImages(overlays, categories, "game", config, log)
	
	// Verify
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
	if size == 0 {
		t.Error("Size is 0")
	}
}

func createDummyJpg(t *testing.T, path string) {
	dc := gg.NewContext(100, 100)
	dc.DrawRectangle(0, 0, 100, 100)
	dc.SetRGB(1, 1, 1)
	dc.Fill()
	// Save as jpg
	// We need jpeg encode.
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	jpeg.Encode(f, dc.Image(), nil)
}

func TestDecodeJpg_InvalidContent(t *testing.T) {
	log, _ := mockLogger()
	// Create invalid file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.jpg")
	os.WriteFile(path, []byte("NOT A JPG"), 0644)
	
	_, err := decodeJpg(path, log)
	if err == nil {
		t.Error("Expected error for invalid jpg content")
	}
	// Check log
	if len(log.Entries) == 0 {
		t.Error("Expected log entry")
	}
}

func TestAddMRCLogo_NilCache(t *testing.T) {
	// Need a context
	dc := gg.NewContext(100, 100)
	
	// Ensure font exists for loadFont to succeed
	// We use project root relative path? 
	// The function uses config.FontsDir.
	// We need to setup a real font file at a known location.
	tmpDir := t.TempDir()
	fontDir := tmpDir
	fontName := "TestFont.ttf"
	createDummyFont(t, filepath.Join(fontDir, fontName))
	
	watermark := &WatermarkData{
		Text: "Test",
		Font: fontName,
		FontSize: 10,
		Location: Point2d{X:0,Y:0},
		BackgroundColour: "#FFFFFF",
		TextColour: "#000000",
	}
	
	// Call with nil cache
	// This will trigger 'else' branch in addMRCLogo which calls loadFont
	// loadFont panics on error, so this also verifies loadFont success path.
	addMRCLogo(dc, watermark, "1.0", "example.com", 0, 0, 1.0, fontDir, nil)
	
	// If no panic, success.
}

// Helper to create dummy font if not exists (reused logic)
func createDummyFont(t *testing.T, dest string) {
	// Try to copy from ../../resources/fonts/Dirga.ttf
	// Adjust path based on where test is running (mrc/common)
	src := "../../resources/fonts/Dirga.ttf"
	data, err := os.ReadFile(src)
	if err != nil {
		// Try creating a minimal valid TTF if source missing? 
		// Or just fail.
		t.Fatalf("Failed to read original font for dummy copy: %v", err)
	}
	os.WriteFile(dest, data, 0644)
}

func TestGenerateImages_MissingLogo(t *testing.T) {
	// Test that GenerateImages handles a missing logo file gracefully
	tmpDir := t.TempDir()
	fontsDir := "../../resources/fonts"
	logoDir := filepath.Join(tmpDir, "logo") // Logo dir exists but no file
	hotasDir := filepath.Join(tmpDir, "hotas")
	os.Mkdir(logoDir, 0755)
	os.Mkdir(hotasDir, 0755)

	// Create dummy device image but no logo
	createDummyJpg(t, filepath.Join(hotasDir, "dev.jpg"))

	config := &Config{
		LogoImagesDir:  logoDir,
		HotasImagesDir: hotasDir,
		FontsDir:       fontsDir,
		InputFont:      "Dirga.ttf",
		InputMinFontSize: 5,
		DefaultImage:   Dimensions2d{W: 100, H: 100},
		PixelMultiplier: 1.0,
		Devices: Devices{
			DeviceLabelsByImage: map[string]string{"dev": "Device"},
			ImageSizeOverride:   make(map[string]Dimensions2d),
		},
	}

	overlays := make(OverlaysByProfile)
	overlays["Default"] = make(OverlaysByImage)
	overlays["Default"]["dev"] = map[string]OverlayData{
		"k1": {PosAndSize: InputData{X: 10, Y: 10, W: 20, H: 20}, ContextToTexts: map[string][]string{"c": {"T"}}},
	}

	categories := map[string]string{"c": "#FFFFFF"}
	log, _ := mockLogger()

	// Run - should return empty due to missing logo
	files, size := GenerateImages(overlays, categories, "missing_game", config, log)

	if len(files) != 0 {
		t.Errorf("Expected 0 files when logo is missing, got %d", len(files))
	}
	if size != 0 {
		t.Errorf("Expected 0 size when logo is missing, got %d", size)
	}
	// Check error was logged
	if len(log.Entries) == 0 {
		t.Error("Expected error log for missing logo")
	}
}

func TestGenerateImages_MissingDeviceImage(t *testing.T) {
	// Test that GenerateImages handles a missing device image gracefully
	tmpDir := t.TempDir()
	fontsDir := "../../resources/fonts"
	logoDir := filepath.Join(tmpDir, "logo")
	hotasDir := filepath.Join(tmpDir, "hotas") // Hotas dir exists but no file
	os.Mkdir(logoDir, 0755)
	os.Mkdir(hotasDir, 0755)

	// Create logo but no device image
	createDummyJpg(t, filepath.Join(logoDir, "game.jpg"))

	config := &Config{
		LogoImagesDir:  logoDir,
		HotasImagesDir: hotasDir,
		FontsDir:       fontsDir,
		InputFont:      "Dirga.ttf",
		InputMinFontSize: 5,
		DefaultImage:   Dimensions2d{W: 100, H: 100},
		PixelMultiplier: 1.0,
		ImageHeader: HeaderData{Font: "Dirga.ttf", FontSize: 14},
		Watermark:   WatermarkData{Font: "Dirga.ttf", FontSize: 10},
		Devices: Devices{
			DeviceLabelsByImage: map[string]string{"missing_dev": "Device"},
			ImageSizeOverride:   make(map[string]Dimensions2d),
		},
	}

	overlays := make(OverlaysByProfile)
	overlays["Default"] = make(OverlaysByImage)
	overlays["Default"]["missing_dev"] = map[string]OverlayData{
		"k1": {PosAndSize: InputData{X: 10, Y: 10, W: 20, H: 20}, ContextToTexts: map[string][]string{"c": {"T"}}},
	}

	categories := map[string]string{"c": "#FFFFFF"}
	log, _ := mockLogger()

	// Run - the goroutine should log error and return early
	files, size := GenerateImages(overlays, categories, "game", config, log)

	// Files slice is pre-allocated, but the entry should be empty
	if len(files) != 1 {
		t.Errorf("Expected 1 file entry in slice, got %d", len(files))
	}
	if files[0].Len() != 0 {
		t.Errorf("Expected empty buffer for missing device image, got %d bytes", files[0].Len())
	}
	if size != 0 {
		t.Errorf("Expected 0 size when device image is missing, got %d", size)
	}
}

func TestAddImageHeader_NilCache(t *testing.T) {
	dc := gg.NewContext(200, 100)

	tmpDir := t.TempDir()
	fontDir := tmpDir
	fontName := "TestFont.ttf"
	createDummyFont(t, filepath.Join(fontDir, fontName))

	header := &HeaderData{
		Font:             fontName,
		FontSize:         14,
		Inset:            Point2d{X: 5, Y: 20},
		BackgroundColour: "#333333",
		BackgroundHeight: 30,
		TextColour:       "#FFFFFF",
	}

	// Call with nil fontCache - should use loadFont directly
	addImageHeader(dc, header, ProfileDefault, "Test Device", 50, 1.0, fontDir, 5, nil)

	// If no panic, success
}

func TestAddImageHeader_CustomProfile(t *testing.T) {
	dc := gg.NewContext(200, 100)

	tmpDir := t.TempDir()
	fontDir := tmpDir
	fontName := "TestFont.ttf"
	createDummyFont(t, filepath.Join(fontDir, fontName))

	header := &HeaderData{
		Font:             fontName,
		FontSize:         14,
		Inset:            Point2d{X: 5, Y: 20},
		BackgroundColour: "#333333",
		BackgroundHeight: 30,
		TextColour:       "#FFFFFF",
	}

	cache := NewFontFaceCache()

	// Call with custom profile (not ProfileDefault) to trigger label formatting
	addImageHeader(dc, header, "CustomProfile", "Test Device", 50, 1.0, fontDir, 5, cache)

	// If no panic, success
}


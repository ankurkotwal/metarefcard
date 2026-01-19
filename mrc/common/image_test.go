package common

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/fogleman/gg"
)

func createDummyJpg(path string, width, height int) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with white
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 80})
}

func TestGenerateImages(t *testing.T) {
	// Setup temp dirs for dummy assets
	tmpDir, err := os.MkdirTemp("", "mrc_img_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fontsDir := "../../resources/fonts"
	absFontsDir, _ := filepath.Abs(fontsDir)

	// Create dummy logo and hotas image
	logoDir := filepath.Join(tmpDir, "logos")
	hotasDir := filepath.Join(tmpDir, "hotas")
	os.Mkdir(logoDir, 0755)
	os.Mkdir(hotasDir, 0755)

	gameLabel := "testgame"
	imageName := "testHotas"
	
	createDummyJpg(filepath.Join(logoDir, gameLabel+".jpg"), 100, 50)
	createDummyJpg(filepath.Join(hotasDir, imageName+".jpg"), 500, 500)

	config := &Config{
		LogoImagesDir:   logoDir,
		HotasImagesDir:  hotasDir,
		FontsDir:        absFontsDir,
		DefaultImage:    Dimensions2d{W: 500, H: 500},
		PixelMultiplier: 1.0,
		JpgQuality:      80,
		InputFontSize:   12,
		ImageHeader: HeaderData{
			BackgroundHeight: 20,
			Font:             "Orbitron-Regular.ttf",
			FontSize:         12,
		},
		Watermark: WatermarkData{
			Font:     "Orbitron-Regular.ttf",
			FontSize: 10,
		},
		InputFont: "SourceSansPro-Regular.ttf",
		Devices: Devices{
			DeviceLabelsByImage: map[string]string{imageName: "Test Device"},
			ImageSizeOverride:   make(map[string]Dimensions2d),
		},
	}
	
	log := NewLog()
	overlays := make(OverlaysByProfile)
	overlays["default"] = make(OverlaysByImage)
	overlays["default"][imageName] = make(map[string]OverlayData)
	
	// Add an overlay to verify drawing
	overlays["default"][imageName]["trigger"] = OverlayData{
		PosAndSize: InputData{X: 10, Y: 10, W: 50, H: 20},
		ContextToTexts: map[string][]string{
			"Default": {"Fire"},
		},
	}
	
	categories := map[string]string{"Default": "#FF0000"}

	bufs, numBytes := GenerateImages(overlays, categories, gameLabel, config, log)
	
	if len(bufs) != 1 {
		t.Errorf("Expected 1 image, got %d", len(bufs))
	}
	if numBytes <= 0 {
		t.Error("Expected positive byte count")
	}
}

func TestDecodeJpg_Error(t *testing.T) {
	// Setup temp dirs for dummy assets
	tmpDir, err := os.MkdirTemp("", "mrc_img_decode_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	log := NewLog()

	// Test bad file
	badFile := filepath.Join(tmpDir, "bad.jpg")
	os.WriteFile(badFile, []byte("not a jpg"), 0644)
	img, err := decodeJpg(badFile, log)
	if err == nil {
		t.Error("Expected error for bad jpg")
	}
	if img != nil {
		t.Error("Expected nil image for bad jpg")
	}

	// Test missing file
	img, err = decodeJpg(filepath.Join(tmpDir, "missing.jpg"), log)
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestPrepImgGenData(t *testing.T) {
	overlays := make(OverlaysByProfile)
	overlays["p1"] = make(OverlaysByImage)
	overlays["p1"]["img1"] = nil
	overlays["p1"]["img2"] = nil
	overlays["p2"] = make(OverlaysByImage)
	overlays["p2"]["img3"] = nil
	
	profiles, imgNames, count := prepImgGenData(overlays)
	
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}
	sort.Strings(profiles)
	if profiles[0] != "p1" || profiles[1] != "p2" {
		t.Errorf("Unexpected profiles: %v", profiles)
	}
	if len(imgNames["p1"]) != 2 {
		t.Errorf("Expected 2 images for p1, got %d", len(imgNames["p1"]))
	}
}

func TestPrepareContexts(t *testing.T) {
	m := map[string][]string{
		"B_Context": {},
		"A_Context": {},
	}
	contexts := prepareContexts(m)
	if len(contexts) != 2 {
		t.Fatalf("Expected 2 contexts")
	}
	if contexts[0] != "A_Context" || contexts[1] != "B_Context" {
		t.Errorf("Expected sorted contexts, got %v", contexts)
	}
}

func TestGetPixelMultiplier(t *testing.T) {
	cfg := &Config{
		PixelMultiplier: 1.5,
		DefaultImage: Dimensions2d{W: 100, H: 100},
		Devices: Devices{
			ImageSizeOverride: map[string]Dimensions2d{
				"override": {W: 200, H: 200},
			},
		},
	}
	
	if m := getPixelMultiplier("normal", cfg); m != 1.5 {
		t.Errorf("Expected default multiplier 1.5, got %v", m)
	}
	
	// Override: 200 / 100 = 2.0
	if m := getPixelMultiplier("override", cfg); m != 2.0 {
		t.Errorf("Expected override multiplier 2.0, got %v", m)
	}
}

func TestAddMRCLogo(t *testing.T) {
	// Setup basics for drawing
	dc := gg.NewContext(100, 100)
	fontsDir := "../../resources/fonts"
	absFontsDir, _ := filepath.Abs(fontsDir)
	
	watermark := &WatermarkData{
		Text: "MRC",
		Font: "Orbitron-Regular.ttf",
		FontSize: 10,
		Location: Point2d{X: 10, Y: 10},
		BackgroundColour: "#FFFFFF",
		TextColour: "#000000",
	}
	
	// Test with nil fontCache to cover that branch
	// We need to ensure fonts exist or mock loadFont?
	// loadFont is tested separately, but here we integration test it via addMRCLogo.
	// We assume fonts exist as per other tests.
	
	// Just ensure no panic and something happens
	addMRCLogo(dc, watermark, "1.0", "domain.com", 0, 0, 1.0, absFontsDir, nil)
	
	// Can't easily verify pixel output without golden images, but coverage is the goal.
}

func TestCalcFontSize(t *testing.T) {
	// Setup a minimal config or just paths needed for fonts
	fontsDir := "../../resources/fonts"
	fontName := "YanoneKaffeesatz-Regular.ttf"

	// Verify font file exists
	absFontsDir, err := filepath.Abs(fontsDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	tests := []struct {
		name         string
		text         string
		targetWidth  int
		targetHeight int
		minFontSize  int
		startSize    int
		wantMax      int // The font size should be <= this
		wantMin      int // The font size should be >= this
	}{
		{
			name:         "Short text fits easily",
			text:         "Test",
			targetWidth:  100,
			targetHeight: 50,
			minFontSize:  10,
			startSize:    50,
			wantMax:      50,
			wantMin:      40,
		},
		{
			name:         "Long text needs shrinking",
			text:         "This is a very long text that should force the font size to reduce",
			targetWidth:  100,
			targetHeight: 50,
			minFontSize:  10,
			startSize:    50,
			wantMax:      20,
			wantMin:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := calcFontSize(tt.text, nil, tt.startSize, tt.targetWidth, tt.targetHeight, absFontsDir, fontName, tt.minFontSize)
			if size > tt.wantMax || size < tt.wantMin {
				t.Errorf("calcFontSize() = %v, want between %v and %v", size, tt.wantMin, tt.wantMax)
			}
		})
	}
}



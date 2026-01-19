package mrc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"html/template"
	"image"
	"image/draw"
	_ "image/jpeg"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

var update = os.Getenv("UPDATE_REFERENCE") == "true"

func TestReferenceFiles(t *testing.T) {
	// Switch to project root to allow config loading to work as expected
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(wd)
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to chdir to project root: %v", err)
	}
	defer os.Chdir(wd) // Restore

	// Load Config with fixed values for consistency (except Version)
	log := common.NewLog()
	var cfg *common.Config
	
	// Load config relative to new CWD (project root)
	common.LoadYaml("config/config.yaml", &cfg, "Config", log)
	
	// Use actual version as requested, but fix Domain for consistency
	// cfg.Version is loaded from config.yaml
	cfg.Domain = "TEST_DOMAIN"
	
	// Load device info
	common.LoadDevicesInfo(cfg.DevicesFile, &cfg.Devices, log)

	testDataDir := "testdata"
	referenceDir := filepath.Join(testDataDir, "reference")
	
	if update {
		if err := os.MkdirAll(referenceDir, 0755); err != nil {
			t.Fatalf("Failed to create reference directory: %v", err)
		}
	}

	// Iterate over games
	for _, gameLoader := range GamesInfo {
		label, _, handler, matchFunc := gameLoader()
		
		gameDir := filepath.Join(testDataDir, label)
		
		err := filepath.WalkDir(gameDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "unsupported" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.Contains(path, "reference") {
				return nil
			}

			// Process file
			t.Run(fmt.Sprintf("%s/%s", label, d.Name()), func(t *testing.T) {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("Failed to read input file: %v", err)
				}

				files := [][]byte{content}
				
				// 1. Handle Request
				gameData, gameBinds, gameDevices, gameContexts, gameLogo := handler(files, cfg, log)
				
				// 2. Populate Overlays
				overlaysByImage := common.PopulateImageOverlays(gameDevices, cfg, log, gameBinds, gameData, matchFunc)
				
				// 3. Generate Images
				generatedImages, _ := common.GenerateImages(overlaysByImage, gameContexts, gameLogo, cfg, log)
				
				// 4. Generate HTML
				htmlOutput := generateHTML(t, generatedImages, projectRoot)
				
				referenceFilename := fmt.Sprintf("%s_%s.html", label, d.Name())
				referencePath := filepath.Join(referenceDir, referenceFilename)
				
				if update {
					if err := os.WriteFile(referencePath, htmlOutput, 0644); err != nil {
						t.Fatalf("Failed to write reference file: %v", err)
					}
				} else {
					expectedHTML, err := os.ReadFile(referencePath)
					if err != nil {
						t.Fatalf("Failed to read reference file %s: %v. Run with UPDATE_REFERENCE=true to create it.", referencePath, err)
					}
					
					// Perform smart comparison
					compareHTMLImages(t, htmlOutput, expectedHTML, cfg, gameLogo, projectRoot)
				}
			})
			return nil
		})
		
		if err != nil {
			t.Errorf("Error walking directory %s: %v", gameDir, err)
		}
	}
}

func compareHTMLImages(t *testing.T, gotHTML, wantHTML []byte, cfg *common.Config, logoName string, projectRoot string) {
	// Extract base64 images from HTML
	gotImages := extractImages(t, gotHTML)
	wantImages := extractImages(t, wantHTML)

	if len(gotImages) != len(wantImages) {
		t.Errorf("Image count mismatch: got %d, want %d", len(gotImages), len(wantImages))
		return
	}

	// Load Game Logo to calculate offset
	logoPath := filepath.Join(projectRoot, cfg.LogoImagesDir, fmt.Sprintf("%s.jpg", logoName))
	logoFile, err := os.Open(logoPath)
	if err != nil {
		t.Fatalf("Failed to open game logo %s: %v", logoPath, err)
	}
	defer logoFile.Close()
	logoImg, _, err := image.Decode(logoFile)
	if err != nil {
		t.Fatalf("Failed to decode game logo: %v", err)
	}
	logoWidth := logoImg.Bounds().Dx()

	for i := 0; i < len(gotImages); i++ {
		compareImagesMaskingWatermark(t, gotImages[i], wantImages[i], cfg, logoWidth, i)
	}
}

func extractImages(t *testing.T, htmlBytes []byte) [][]byte {
	// Pattern to capture the base64 content
	ptn := regexp.MustCompile(`data:image/jpg;base64,([^"]+)`)
	matches := ptn.FindAllSubmatch(htmlBytes, -1)
	var images [][]byte
	for _, m := range matches {
		b64 := string(m[1])
		// Unescape HTML entities (e.g. &#43; -> +)
		b64 = html.UnescapeString(b64)
		
		// Strip newlines/whitespace if any
		b64 = strings.ReplaceAll(b64, "\n", "")
		b64 = strings.ReplaceAll(b64, "\r", "")
		b64 = strings.ReplaceAll(b64, " ", "")
		
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			// Print snippet for debugging if fail
			snippet := b64
			if len(snippet) > 50 {
				snippet = snippet[:50]
			}
			t.Fatalf("Failed to decode base64 image (start: %s...): %v", snippet, err)
		}
		images = append(images, data)
	}
	return images
}

func compareImagesMaskingWatermark(t *testing.T, gotBytes, wantBytes []byte, cfg *common.Config, logoWidth int, idx int) {
	gotImg, _, err := image.Decode(bytes.NewReader(gotBytes))
	if err != nil {
		t.Fatalf("Failed to decode got image %d: %v", idx, err)
	}
	wantImg, _, err := image.Decode(bytes.NewReader(wantBytes))
	if err != nil {
		t.Fatalf("Failed to decode want image %d: %v", idx, err)
	}

	bounds := gotImg.Bounds()
	if !bounds.Eq(wantImg.Bounds()) {
		t.Errorf("Image %d bounds mismatch: got %v, want %v", idx, bounds, wantImg.Bounds())
		return
	}
	
	// Convert to RGBA for pixel access
	gotRGBA := ensureRGBA(gotImg)
	wantRGBA := ensureRGBA(wantImg)

	// Calculate Watermark Area to Ignore
	// Based on common/image.go logic
	// The watermark is drawn relative to pixelMultiplier. 
	// We don't easily know the pixelMultiplier per image here (it varies by device), 
	// but we can infer it or just blank out a generous area.
	// Most images use cfg.PixelMultiplier (0.5).
	// Let's assume the watermark is definitely in the header region.
	// Header is xOffset onwards.
	// Watermark Y is around 136 * multiplier.
	// Logo Width (xOffset) is fixed per game.
	
	// We will blank out a strip at the calculated Y level across the "text area" of the header.
	// To be robust, we mask the specific estimated rectangle.
	
	// We need the multiplier. We can try to guess it from image width?
	// Width = DefaultImage.W * multiplier.
	// multiplier = Width / DefaultImage.W.
	
	multiplier := float64(bounds.Dx()) / float64(cfg.DefaultImage.W)
	
	// Watermark Location
	// x := xOffset + (location.X)*pixelMultiplier
	// y := (location.Y) * pixelMultiplier
	// But note: drawTextWithBackgroundRec does fancy centering.
	// Let's be aggressive: Mask the Watermark line.
	
	maskX := float64(logoWidth) + float64(cfg.Watermark.Location.X)*multiplier
	maskY := float64(cfg.Watermark.Location.Y) * multiplier
	
	// Approximate height/width of watermark
	// Font size 44.
	maskH := 60.0 * multiplier 
	// maskW := float64(bounds.Dx()) // Mask till end of screen to be safe
	
	// Adjust Y to account for centering/padding fuzzy logic in image.go
	// It centers in targetHeight? No, it uses FontSize as targetHeight.
	// So Y starts roughly at location.Y.
	
	minY := int(maskY - 10) // buffer
	maxY := int(maskY + maskH + 10)
	minX := int(maskX - 10)
	
	// Compare pixels
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Skip if in mask
			if x >= minX && y >= minY && y <= maxY {
				continue
			}
			
			c1 := gotRGBA.RGBAAt(x, y)
			c2 := wantRGBA.RGBAAt(x, y)
			
			if c1 != c2 {
				t.Errorf("Image %d pixel mismatch at (%d, %d). Got %v, Want %v", idx, x, y, c1, c2)
				return // Fail fast per image
			}
		}
	}
}

func ensureRGBA(img image.Image) *image.RGBA {
	if dst, ok := img.(*image.RGBA); ok {
		return dst
	}
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, img, b.Min, draw.Src)
	return dst
}

func generateHTML(t *testing.T, generatedFiles []bytes.Buffer, projectRoot string) []byte {
	// Replicating logic from sendResponse for HTML generation
	// Use relative path from project root
	cardTempl := "resources/www/templates/refcard.html"
	tmpl, err := template.New("refcard.html").ParseFiles(cardTempl)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	type base64Image struct {
		Base64Contents string
	}

	var fullOutput bytes.Buffer

	for _, file := range generatedFiles {
		image := base64Image{
			Base64Contents: base64.StdEncoding.EncodeToString(file.Bytes()),
		}
		if err := tmpl.Execute(&fullOutput, image); err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
	}
	
	// Note: We are NOT including the log output in the reference file comparison 
	// because logs contain timestamps and variable pointers that change.
	
	return fullOutput.Bytes()
}

package common

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/fogleman/gg"
)

// TestAllControllerImages_RenderFieldsWithinBounds loads the production config
// and device definitions, renders text into every configured input field on
// every controller image, and verifies that:
//  1. Every field's bounding box lies within the image dimensions.
//  2. Every rasterized text pill (both single-label and multi-binding long text)
//     stays strictly within its field's bounding box (within 1px anti-aliasing
//     tolerance).
//  3. The full reference card (header, watermark, and all fields) renders and
//     encodes cleanly as a JPEG.
//
// Set MRC_TEST_OUTPUT_DIR=/path/to/dir to write each rendered controller card
// JPEG for visual inspection.
func TestAllControllerImages_RenderFieldsWithinBounds(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("failed to chdir to repo root %s: %v", repoRoot, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	log := NewLog()
	log.FatalFunc = func(format string, v ...interface{}) {
		t.Fatalf(format, v...)
	}

	var config Config
	LoadYaml(filepath.Join("config", "config.yaml"), &config, "Config", log)
	LoadDevicesInfo(config.DevicesFile, &config.Devices, log)

	// Group device shortNames by imageName.
	devicesByImage := make(map[string][]string)
	for devName, imgName := range config.Devices.ImageMap {
		devicesByImage[imgName] = append(devicesByImage[imgName], devName)
	}

	imageNames := make([]string, 0, len(devicesByImage))
	for imgName := range devicesByImage {
		imageNames = append(imageNames, imgName)
		sort.Strings(devicesByImage[imgName])
	}
	sort.Strings(imageNames)

	outputDir := os.Getenv("MRC_TEST_OUTPUT_DIR")
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Fatalf("failed to create MRC_TEST_OUTPUT_DIR %s: %v", outputDir, err)
		}
	}

	// Build a deterministic category palette from config.AlternateColours.
	categories := make(map[string]string, len(config.AlternateColours))
	numColours := len(config.AlternateColours)
	if numColours > 6 {
		// Use the first 6 full-opacity colours for crisp visual readability.
		numColours = 6
	}
	for i := 0; i < numColours; i++ {
		categories[fmt.Sprintf("cat%d", i)] = config.AlternateColours[i]
	}

	fontCache := NewFontFaceCache()

	for _, imgName := range imageNames {
		imgName := imgName
		devNames := devicesByImage[imgName]

		t.Run(imgName, func(t *testing.T) {
			// Collect unique valid input fields across all devices sharing this image.
			type fieldSpec struct {
				key   string
				input InputData
			}
			seenBox := make(map[InputData]bool)
			var fields []fieldSpec

			for _, devName := range devNames {
				inputs, ok := config.Devices.Index[devName]
				if !ok {
					continue
				}
				keys := make([]string, 0, len(inputs))
				for k := range inputs {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					in := inputs[k]
					if in.X == -1 || in.Y == -1 || in.W <= 0 || in.H <= 0 {
						continue
					}
					if seenBox[in] {
						continue
					}
					seenBox[in] = true
					fields = append(fields, fieldSpec{key: k, input: in})
				}
			}

			if len(fields) == 0 {
				t.Skipf("image %q has no DeviceMap input fields configured (devices: %v)", imgName, devNames)
				return
			}

			imageFilename := filepath.Join(config.HotasImagesDir, imgName+".jpg")
			baseImg, err := decodeJpg(imageFilename, log)
			if err != nil {
				t.Fatalf("failed to decode controller image %s: %v", imageFilename, err)
			}
			imgW := baseImg.Bounds().Dx()
			imgH := baseImg.Bounds().Dy()
			pixelMultiplier := getPixelMultiplier(imgName, &config)

			// Reuse a single RGBA canvas across all fields for this controller image.
			scratchImg := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
			fieldDC := gg.NewContextForRGBA(scratchImg)

			overlays := make(map[string]OverlayData, len(fields))
			catIdx := 0

			for _, f := range fields {
				label := formatFieldSampleText(f.key)
				catKey := fmt.Sprintf("cat%d", catIdx%numColours)
				catIdx++

				overlays[f.key] = OverlayData{
					PosAndSize: f.input,
					ContextToTexts: map[string][]string{
						catKey: {label},
					},
				}

				// 1. Verify the field's bounding box lies within the image.
				boxMinX := float64(f.input.X) * pixelMultiplier
				boxMinY := float64(f.input.Y) * pixelMultiplier
				boxMaxX := float64(f.input.X+f.input.W) * pixelMultiplier
				boxMaxY := float64(f.input.Y+f.input.H) * pixelMultiplier

				if boxMinX < 0 || boxMinY < 0 || boxMaxX > float64(imgW)+1 || boxMaxY > float64(imgH)+1 {
					t.Errorf("field %q box [%.1f, %.1f, %.1f, %.1f] exceeds image bounds %dx%d (multiplier=%.4f, raw=%+v)",
						f.key, boxMinX, boxMinY, boxMaxX, boxMaxY, imgW, imgH, pixelMultiplier, f.input)
					continue
				}

				// 2. Render both single-label and multi-binding text onto the reused
				//    transparent canvas and verify every rasterized pixel stays inside
				//    the field's bounding box.
				testCases := []struct {
					name  string
					texts []string
				}{
					{name: "single", texts: []string{label}},
					{name: "multi", texts: []string{label, "Secondary Action"}},
				}

				for _, tc := range testCases {
					clear(scratchImg.Pix)
					verifyRenderedFieldBounds(t, fieldDC, scratchImg, f.key, tc.name, tc.texts,
						f.input, pixelMultiplier, categories[catKey], &config, fontCache)
				}
			}

			// 3. Render the complete card (logo, header, watermark, all fields) via GenerateImages.
			overlaysByProfile := OverlaysByProfile{
				ProfileDefault: OverlaysByImage{
					imgName: overlays,
				},
			}
			cardLog := NewLog()
			cardLog.FatalFunc = log.FatalFunc
			files, totalBytes := GenerateImages(overlaysByProfile, categories, "fs2020", &config, cardLog)
			if len(files) != 1 || totalBytes == 0 {
				t.Fatalf("GenerateImages failed for %q: files=%d totalBytes=%d", imgName, len(files), totalBytes)
			}
			for _, entry := range cardLog.Entries {
				if entry.IsError {
					t.Errorf("GenerateImages logged error for %q: %s", imgName, entry.Msg)
				}
			}

			if outputDir != "" {
				outPath := filepath.Join(outputDir, imgName+".jpg")
				if err := os.WriteFile(outPath, files[0].Bytes(), 0644); err != nil {
					t.Errorf("failed to write output image %s: %v", outPath, err)
				}
			}
		})
	}
}

func verifyRenderedFieldBounds(t *testing.T, dc *gg.Context, rgba *image.RGBA,
	fieldKey string, caseName string, texts []string, input InputData,
	pixelMultiplier float64, bgColor string, config *Config, fontCache FontLoader) {
	t.Helper()

	fontSize := int(math.Round(config.InputFontSize * pixelMultiplier))
	targetWidth := int(math.Round((float64(input.W) - 2*config.InputPixelXInset) * pixelMultiplier))
	targetHeight := int(math.Round((float64(input.H) - 2*config.InputPixelYInset) * pixelMultiplier))

	fullText := ""
	incrementalTexts := []string{""}
	for _, text := range texts {
		if len(fullText) != 0 {
			fullText = fmt.Sprintf("%s %s", fullText, text)
		} else {
			fullText = text
		}
		incrementalTexts = append(incrementalTexts, fullText+" ")
	}

	fontSize = calcFontSize(fullText, fontCache, fontSize, targetWidth,
		targetHeight, config.FontsDir, config.InputFont, config.InputMinFontSize)

	largeFont := fontCache.LoadFont(config.FontsDir, config.InputFont, fontSize)
	smallFont := fontCache.LoadFont(config.FontsDir, config.InputFont, fontSize-1)

	for i, text := range texts {
		offset, _ := measureString(largeFont, incrementalTexts[i])
		drawTextWithBackgroundRec(dc, text, float64(offset),
			Point2d{X: float64(input.X), Y: float64(input.Y)},
			config.InputPixelXInset, config.InputPixelYInset,
			targetHeight, pixelMultiplier, largeFont, smallFont,
			bgColor, config.LightColour)
	}

	drawnRect, hasPixels := nonTransparentBounds(rgba)
	if !hasPixels {
		t.Errorf("field %q (%s) rendered no visible pixels (texts=%q)", fieldKey, caseName, texts)
		return
	}

	const tol = 1.0 // 1px tolerance for subpixel rounding & anti-aliasing
	boxMinX := float64(input.X) * pixelMultiplier
	boxMinY := float64(input.Y) * pixelMultiplier
	boxMaxX := float64(input.X+input.W) * pixelMultiplier
	boxMaxY := float64(input.Y+input.H) * pixelMultiplier

	allowedMinX := int(math.Floor(boxMinX - tol))
	allowedMinY := int(math.Floor(boxMinY - tol))
	allowedMaxX := int(math.Ceil(boxMaxX + tol))
	allowedMaxY := int(math.Ceil(boxMaxY + tol))

	if drawnRect.Min.X < allowedMinX || drawnRect.Min.Y < allowedMinY ||
		drawnRect.Max.X > allowedMaxX || drawnRect.Max.Y > allowedMaxY {
		t.Errorf("field %q (%s) rendered pixels %v outside bounding box [%d,%d]-[%d,%d] (texts=%q, fontSize=%d, raw=%+v)",
			fieldKey, caseName, drawnRect, allowedMinX, allowedMinY, allowedMaxX, allowedMaxY, texts, fontSize, input)
	}
}

func formatFieldSampleText(key string) string {
	if _, err := strconv.Atoi(key); err == nil {
		return "Button " + key
	}
	return key
}

func nonTransparentBounds(rgba *image.RGBA) (image.Rectangle, bool) {
	b := rgba.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X-1, b.Min.Y-1

	for y := b.Min.Y; y < b.Max.Y; y++ {
		rowOff := (y - rgba.Rect.Min.Y) * rgba.Stride
		for x := b.Min.X; x < b.Max.X; x++ {
			if rgba.Pix[rowOff+(x-rgba.Rect.Min.X)*4+3] > 0 {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if maxX < minX || maxY < minY {
		return image.Rectangle{}, false
	}
	return image.Rect(minX, minY, maxX+1, maxY+1), true
}

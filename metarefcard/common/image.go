package common

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"math"
	"sort"
	"sync"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// GenerateImage - generates an image with the provided overlays
func GenerateImage(dc *gg.Context, image *image.Image, imageFilename string,
	overlaysByImage OverlaysByImage, categories map[string]string,
	config *Config) *bytes.Buffer {

	// Set the background colour
	dc.SetHexColor(config.BackgroundColour)
	dc.Clear()
	// Apply the image on top
	dc.DrawImage(*image, 0, 0)
	dc.SetRGB(0, 0, 0)
	pixelMultiplier := getPixelMultiplier(imageFilename, dc, config)

	overlayDataRange := overlaysByImage[imageFilename]
	for _, overlayData := range overlayDataRange {
		// Skip known bad locations
		if overlayData.PosAndSize.X == -1 || overlayData.PosAndSize.Y == -1 {
			continue
		}
		xLoc := float64(overlayData.PosAndSize.X) * pixelMultiplier
		yLoc := float64(overlayData.PosAndSize.Y) * pixelMultiplier

		if xLoc >= float64(dc.Width()) || yLoc >= float64(dc.Height()) {
			log.Printf("Error: Overlay outside bounds. File %s overlayData %v defaults %v\n",
				imageFilename, overlayData.PosAndSize, config.DefaultImage)
			continue
		}

		fontSize := int(math.Round(float64(config.InputFontSize) * pixelMultiplier))
		targetWidth := int(math.Round(float64(overlayData.PosAndSize.W-config.InputPixelInset) * pixelMultiplier))
		targetHeight := int(math.Round(float64(overlayData.PosAndSize.H) * pixelMultiplier))

		// Iterate through contexts (in order) and texts (already sorted)
		// to generate text to be displayed
		fullText := ""
		incrementalTexts := []string{""}
		for _, context := range prepareContexts(overlayData.ContextToTexts) {
			texts := overlayData.ContextToTexts[context]
			// First get the full text to workout font size
			for _, text := range texts {
				padding := " "
				if len(fullText) != 0 {
					fullText = fmt.Sprintf("%s%s%s", fullText, padding, text)
				} else {
					fullText = text
				}
				incrementalTexts = append(incrementalTexts, fullText+padding)
			}
		}
		fontSize = calcFontSize(fullText, fontSize, targetWidth, targetHeight, config)
		// Now create overlays for each text
		// Ugh, second loop through texts
		idx := 0
		for _, context := range prepareContexts(overlayData.ContextToTexts) {
			texts := overlayData.ContextToTexts[context]
			for _, text := range texts {
				offset, _ := measureString(getFontBySize(fontSize, config), incrementalTexts[idx])
				idx++

				x := offset + int(math.Round(float64(overlayData.PosAndSize.X+config.InputPixelInset)*pixelMultiplier))
				y := int(math.Round(float64(overlayData.PosAndSize.Y) * pixelMultiplier))
				w, h := measureString(getFontBySize(fontSize, config), text)
				// Vertically center
				y = y + (targetHeight-h)/2

				dc.SetHexColor(categories[context])
				dc.DrawRoundedRectangle(float64(x), float64(y), float64(w), float64(h), 6)
				dc.Fill()
				dc.SetHexColor(config.LightColour)
				face := getFontBySize(fontSize, config)
				dc.SetFontFace(face) // Render one font size smaller to fit in rect
				w2, _ := measureString(face, text)
				dc.DrawStringAnchored(text, float64(x+(w-w2)/2), float64(y), 0, 0.85)
			}
		}
	}
	var imgBytes bytes.Buffer
	dc.EncodeJPG(&imgBytes, &jpeg.Options{Quality: 90})
	return &imgBytes
}

var fontBySize map[int]font.Face = make(map[int]font.Face)
var fontsMux sync.Mutex

func getFontBySize(size int, config *Config) font.Face {
	fontsMux.Lock()
	face, found := fontBySize[size]
	if !found {
		name := fmt.Sprintf("%s/%s", config.FontsDir, config.InputFont)

		fontBytes, err := ioutil.ReadFile(name)
		if err != nil {
			panic(err)
		}
		f, err := truetype.Parse(fontBytes)
		if err != nil {
			panic(err)
		}
		face = truetype.NewFace(f, &truetype.Options{
			Size: float64(size),
		})
		fontBySize[size] = face
	}
	fontsMux.Unlock()
	return face
}

var measureMux sync.Mutex

func measureString(fontFace font.Face, text string) (int, int) {
	measureMux.Lock()
	calcX := font.MeasureString(fontFace, text).Round()
	calcY := fontFace.Metrics().Height.Round()
	measureMux.Unlock()
	return calcX, calcY
}

// Resize font till it fits
func calcFontSize(text string, fontSize int, targetWidth int, targetHeight int,
	config *Config) int {
	// Max height in pixels is targetHeight (fontSize = height)
	maxFontSize := targetHeight
	minFontSize := 13
	newFontSize := maxFontSize
	for {
		x, y := measureString(getFontBySize(newFontSize, config), text)
		if y > targetHeight {
			panic("Text is taller than max height")
		}
		if x > targetWidth {
			// Need to reduce fontSize and try again
			maxFontSize = newFontSize - 1
			delta := ((newFontSize - minFontSize) / 2)
			if delta == 0 {
				// Found optimal size
				break
			}
			newFontSize -= delta
		} else {
			if newFontSize == maxFontSize || newFontSize == maxFontSize-1 {
				// We optimally fit in the space, we're done.
				break
			}
			// Can grow more, do so.
			minFontSize = newFontSize + 1
			delta := ((maxFontSize - newFontSize) / 2)
			if delta == 0 {
				// Found optimal size
				break
			}
			newFontSize += delta
		}
	}
	return newFontSize
}

func prepareContexts(contextToTexts map[string][]string) []string {
	// Get a list of contexts and sort them
	contexts := make([]string, 0, len(contextToTexts))
	for context := range contextToTexts {
		contexts = append(contexts, context)
	}
	sort.Strings(contexts)
	return contexts
}

// Return the multiplier/scale of image based on actual width vs default width
func getPixelMultiplier(name string, dc *gg.Context, config *Config) float64 {
	multiplier := config.PixelMultiplier
	if dimensions, found := config.ImageSizeOverride[name]; found {
		multiplier = float64(dimensions.W) / float64(config.DefaultImage.W)
	}
	return multiplier
}

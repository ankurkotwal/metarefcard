package common

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"sort"
	"sync"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

var waterMarkFont *font.Face = nil
var headingFont *font.Face = nil
var gameLogos map[string]image.Image = make(map[string]image.Image)

// GenerateImage - generates an image with the provided overlays
func GenerateImage(dc *gg.Context, image *image.Image, imageFilename string,
	overlaysByImage OverlaysByImage, categories map[string]string,
	config *Config, log *Logger, gameLogo string) *bytes.Buffer {

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
			log.Err("Overlay outside bounds. File %s overlayData %v defaults %v",
				imageFilename, overlayData.PosAndSize, config.DefaultImage)
			continue
		}

		fontSize := int(math.Round(config.InputFontSize * pixelMultiplier))
		targetWidth := int(math.Round(float64(overlayData.PosAndSize.W-2*config.InputPixelXInset) * pixelMultiplier))
		targetHeight := int(math.Round(float64(overlayData.PosAndSize.H-2*config.InputPixelYInset) * pixelMultiplier))

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

				x := offset + int(math.Round(float64(overlayData.PosAndSize.X+config.InputPixelXInset)*pixelMultiplier))
				y := int(math.Round(float64(overlayData.PosAndSize.Y+config.InputPixelYInset) * pixelMultiplier))
				w, h := measureString(getFontBySize(fontSize, config), text)
				// Vertically center
				y += (targetHeight - h) / 2

				dc.SetHexColor(categories[context])
				dc.DrawRoundedRectangle(float64(x), float64(y), float64(w), float64(h), 6)
				dc.Fill()
				dc.SetHexColor(config.LightColour)
				// Decrease font size to fit nicely in the rectangle
				face := getFontBySize(fontSize-1, config)
				dc.SetFontFace(face) // Render one font size smaller to fit in rect
				w2, h2 := measureString(face, text)
				dc.DrawStringAnchored(text, float64(x+(w-w2)/2), float64(y+(h-h2)/2), 0, 0.83)
			}
		}
	}

	// Add game logo
	logo, found := gameLogos[gameLogo]
	if !found {
		var err error
		logo, err = gg.LoadImage(fmt.Sprintf("%s/%s.png", config.LogoImagesDir, gameLogo))
		if err != nil {
			log.Err("loadImage %s failed. %v", imageFilename, err)
		}
		gameLogos[gameLogo] = logo
	}
	dc.DrawImage(logo, 0, 0)
	xOffset := float64(logo.Bounds().Max.X)

	// Generate Heading
	dc.SetHexColor(config.ImageHeading.BackgroundColour)
	dc.DrawRectangle(xOffset, 0, float64(dc.Width())-xOffset,
		config.ImageHeading.BackgroundHeight*pixelMultiplier)
	dc.Fill()
	if headingFont == nil {
		headingFont = LoadFont(config.FontsDir, config.ImageHeading.Font,
			int(math.Round(config.ImageHeading.FontSize*pixelMultiplier)))
	}
	dc.SetHexColor(config.ImageHeading.TextColour)
	dc.SetFontFace(*headingFont)
	dc.DrawString(fmt.Sprintf("%s", config.Devices.DeviceLabelsByImage[imageFilename]),
		xOffset+config.ImageHeading.Inset.X*pixelMultiplier,
		config.ImageHeading.Inset.Y*pixelMultiplier)
	// Generate watermark
	if waterMarkFont == nil {
		waterMarkFont = LoadFont(config.FontsDir, config.Watermark.Font,
			int(math.Round(config.Watermark.FontSize*pixelMultiplier)))
	}
	dc.SetHexColor(config.Watermark.TextColour)
	dc.SetFontFace(*waterMarkFont)
	dc.DrawString(fmt.Sprintf("%s (%s)", config.Watermark.Text, config.Version),
		xOffset+config.Watermark.Location.X*pixelMultiplier,
		config.Watermark.Location.Y*pixelMultiplier)

	var imgBytes bytes.Buffer
	jpeg.Encode(&imgBytes, dc.Image(), &jpeg.Options{Quality: config.JpgQuality})
	return &imgBytes
}

var fontBySize map[int]font.Face = make(map[int]font.Face)
var fontsMux sync.Mutex

func getFontBySize(size int, config *Config) font.Face {
	fontsMux.Lock()
	face, found := fontBySize[size]
	if !found {
		face = *LoadFont(config.FontsDir, config.InputFont, size)
		fontBySize[size] = face
	}
	fontsMux.Unlock()
	return face
}

var fontMuxes map[*font.Face]*sync.Mutex = make(map[*font.Face]*sync.Mutex)

func measureString(fontFace font.Face, text string) (int, int) {
	mux, found := fontMuxes[&fontFace]
	if !found {
		mux = &sync.Mutex{}
		fontMuxes[&fontFace] = mux
	}
	mux.Lock()
	calcX := font.MeasureString(fontFace, text).Round()
	calcY := fontFace.Metrics().Height.Round()
	mux.Unlock()
	return calcX, calcY
}

// Resize font till it fits
func calcFontSize(text string, fontSize int, targetWidth int, targetHeight int,
	config *Config) int {
	// Max height in pixels is targetHeight (fontSize = height)
	maxFontSize := targetHeight
	minFontSize := config.InputMinFontSize
	newFontSize := maxFontSize
	for {
		x, y := measureString(getFontBySize(newFontSize, config), text)
		if y > targetHeight {
			panic("Text is taller than max height")
		}
		if x > targetWidth {
			// Need to reduce fontSize and try again
			maxFontSize = newFontSize - 1
			delta := (newFontSize - minFontSize) / 2
			if delta == 0 {
				// We're too big but can't go between min and current font size,
				// return the min font size
				if x > targetWidth {
					newFontSize = minFontSize
				}
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
			minFontSize = newFontSize
			delta := (maxFontSize - newFontSize) / 2
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
	if dimensions, found := config.Devices.ImageSizeOverride[name]; found {
		multiplier = float64(dimensions.W) / float64(config.DefaultImage.W)
	}
	return multiplier
}

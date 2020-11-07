package common

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	"github.com/fogleman/gg"
	"github.com/pixiv/go-libjpeg/jpeg"
	"golang.org/x/image/font"
)

// GenerateImage - generates an image with the provided overlays
func GenerateImage(imageName string, imageFilename string, profile string,
	overlayDataRange map[string]OverlayData, categories map[string]string,
	config *Config, log *Logger, gameLogo string) bytes.Buffer {

	image, err := gg.LoadImage(fmt.Sprintf("%s/%s.png", config.HotasImagesDir, imageName))
	if err != nil {
		log.Err("loadImage %s failed. %v", imageName, err)
	}

	dc := gg.NewContextForImage(image)
	pixelMultiplier := getPixelMultiplier(imageFilename, config)

	width := float64(image.Bounds().Size().X)
	height := float64(image.Bounds().Size().Y)
	fontFaceCache := make(fontFaceCache)
	for _, overlayData := range overlayDataRange {
		// Skip known bad locations
		if overlayData.PosAndSize.X == -1 || overlayData.PosAndSize.Y == -1 {
			continue
		}
		xLoc := float64(overlayData.PosAndSize.X) * pixelMultiplier
		yLoc := float64(overlayData.PosAndSize.Y) * pixelMultiplier

		if xLoc >= width || yLoc >= height {
			log.Err("Overlay outside bounds. File %s overlayData %v defaults %v",
				imageFilename, overlayData.PosAndSize, config.DefaultImage)
			continue
		}

		fontSize := int(math.Round(config.InputFontSize * pixelMultiplier))
		targetWidth := int(math.Round((float64(overlayData.PosAndSize.W) -
			2*config.InputPixelXInset) * pixelMultiplier))
		targetHeight := int(math.Round((float64(overlayData.PosAndSize.H) -
			2*config.InputPixelYInset) * pixelMultiplier))

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
		fontSize = calcFontSize(fullText, fontFaceCache, fontSize, targetWidth, targetHeight,
			config.FontsDir, config.InputFont, config.InputMinFontSize)
		// Now create overlays for each text
		// Ugh, second loop through texts
		idx := 0
		for _, context := range prepareContexts(overlayData.ContextToTexts) {
			texts := overlayData.ContextToTexts[context]
			for _, text := range texts {
				largeFont := fontFaceCache.loadFont(config.FontsDir, config.InputFont, fontSize)
				smallFont := fontFaceCache.loadFont(config.FontsDir, config.InputFont, fontSize-1)
				offset, _ := measureString(largeFont, incrementalTexts[idx])
				idx++
				location := Point2d{X: float64(overlayData.PosAndSize.X),
					Y: float64(overlayData.PosAndSize.Y)}
				drawTextWithBackgroundRec(dc, text, float64(offset),
					location, config.InputPixelXInset, config.InputPixelYInset,
					targetHeight, pixelMultiplier, largeFont, smallFont,
					categories[context], config.LightColour)
			}
		}
	}

	xOffset := addGameLogo(dc, gameLogo, config.LogoImagesDir, imageFilename, log)
	addImageHeader(dc, &config.ImageHeader, profile,
		config.Devices.DeviceLabelsByImage[imageFilename],
		xOffset, pixelMultiplier, config.FontsDir, config.InputMinFontSize)
	addMRCLogo(dc, &config.Watermark, config.Version,
		xOffset, float64(config.InputPixelXInset), pixelMultiplier, config.FontsDir)

	var imgBytes bytes.Buffer
	jpeg.Encode(&imgBytes, dc.Image(), &jpeg.EncoderOptions{Quality: config.JpgQuality})
	return imgBytes
}

func measureString(fontFace font.Face, text string) (int, int) {
	calcX := font.MeasureString(fontFace, text).Round()
	calcY := fontFace.Metrics().Height.Round()
	return calcX, calcY
}

// Resize font till it fits
func calcFontSize(text string, fontFaceCache fontFaceCache,
	fontSize int, targetWidth int, targetHeight int, fontsDir string,
	fontName string, minFontSize int) int {
	// Max height in pixels is targetHeight (fontSize = height)
	maxFontSize := targetHeight
	newFontSize := maxFontSize
	for {
		var fontFace font.Face
		if fontFaceCache == nil {
			fontFace = loadFont(fontsDir, fontName, newFontSize)
		} else {
			fontFace = fontFaceCache.loadFont(fontsDir, fontName, newFontSize)
		}
		x, y := measureString(fontFace, text)
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
func getPixelMultiplier(name string, config *Config) float64 {
	multiplier := config.PixelMultiplier
	if dimensions, found := config.Devices.ImageSizeOverride[name]; found {
		multiplier = float64(dimensions.W) / float64(config.DefaultImage.W)
	}
	return multiplier
}

func drawTextWithBackgroundRec(dc *gg.Context, text string, xOffset float64,
	location Point2d, xInset float64, yInset float64, targetHeight int,
	pixelMultiplier float64, largeFont font.Face, smallFont font.Face,
	backgroundColour string, textColour string) {
	x := xOffset + (location.X+xInset)*pixelMultiplier
	y := (location.Y + yInset) * pixelMultiplier
	w, h := measureString(largeFont, text)
	// Vertically center
	y += float64(targetHeight-h) / 2
	w2, h2 := measureString(smallFont, text)

	dc.SetHexColor(backgroundColour)
	dc.DrawRoundedRectangle(x, y, float64(w), float64(h), 6)
	dc.Fill()
	dc.SetHexColor(textColour)
	// Decrease font size to fit nicely in the rectangle
	dc.SetFontFace(smallFont) // Render one font size smaller to fit in rect
	dc.DrawStringAnchored(text, x+float64(w-w2)/2, y+float64(h-h2)/2, 0, 0.83)
}

func addGameLogo(dc *gg.Context, gameLogo string, logoImagesDir string,
	imageFilename string, log *Logger) float64 {
	// Add game logo
	logo, err := gg.LoadImage(fmt.Sprintf("%s/%s.png", logoImagesDir, gameLogo))
	if err != nil {
		log.Err("loadImage %s failed. %v", imageFilename, err)
	}
	dc.DrawImage(logo, 0, 0)
	return float64(logo.Bounds().Max.X)
}

func addImageHeader(dc *gg.Context, imageHeader *HeaderData, profile string,
	label string, xOffset float64, pixelMultiplier float64, fontsDir string,
	minFontSize int) {
	fontSize := int(math.Round(imageHeader.FontSize * pixelMultiplier))
	// Add profile name to header if its not the MRC default
	if profile != ProfileDefault {
		label = fmt.Sprintf("%s (%s)", label, profile)
	}
	targetWidth := dc.Width() - int(math.Round(xOffset+2*imageHeader.Inset.X*pixelMultiplier))
	targetHeight := fontSize // Use fontSize as the targetHeight (max height)
	fontSize = calcFontSize(label, nil, fontSize, targetWidth, targetHeight,
		fontsDir, imageHeader.Font, minFontSize)
	headingFont := loadFont(fontsDir, imageHeader.Font, fontSize)

	// Generate header
	dc.SetHexColor(imageHeader.BackgroundColour)
	dc.DrawRectangle(xOffset, 0, float64(dc.Width())-xOffset,
		imageHeader.BackgroundHeight*pixelMultiplier)
	dc.Fill()
	dc.SetHexColor(imageHeader.TextColour)
	dc.SetFontFace(headingFont)
	dc.DrawString(label, xOffset+imageHeader.Inset.X*pixelMultiplier,
		imageHeader.Inset.Y*pixelMultiplier)
}

func addMRCLogo(dc *gg.Context, watermark *WatermarkData, version string,
	xOffset float64, xInset float64, pixelMultiplier float64, fontsDir string) {
	fontSize := int(math.Round(watermark.FontSize * pixelMultiplier))
	// Generate watermark
	text := fmt.Sprintf("%s v%s", watermark.Text, version)
	drawTextWithBackgroundRec(dc, text, xOffset, watermark.Location, 0, 0,
		fontSize, pixelMultiplier,
		loadFont(fontsDir, watermark.Font, fontSize),
		loadFont(fontsDir, watermark.Font, fontSize-1),
		watermark.BackgroundColour, watermark.TextColour)
}

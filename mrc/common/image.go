package common

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"sort"

	"sync"
	"sync/atomic"

	"github.com/fogleman/gg"
	"github.com/pixiv/go-libjpeg/jpeg"
	"golang.org/x/image/font"
)

// GenerateImages returns the generated images
func GenerateImages(overlaysByProfile OverlaysByProfile,
	categories map[string]string,
	gameLabel string, config *Config, log *Logger) ([]bytes.Buffer, int) {

	profiles, imageNamesByProfile, numFiles := prepImgGenData(overlaysByProfile)
	files := make([]bytes.Buffer, 0, numFiles)
	var numBytes int

	// Add game logo
	logoFilename := fmt.Sprintf("%s/%s.jpg", config.LogoImagesDir, gameLabel)
	logo, err := decodeJpg(logoFilename, log)
	if err != nil {
		log.Err("loadImage %s failed. %v", logoFilename, err)
		return files, numBytes
	}

	// Pre-calculate inputs to allow using index for deterministic output order
	type workItem struct {
		profile   string
		imageName string
		index     int
	}
	var workItems []workItem
	idx := 0
	for _, profile := range profiles {
		for _, imageName := range imageNamesByProfile[profile] {
			workItems = append(workItems, workItem{
				profile:   profile,
				imageName: imageName,
				index:     idx,
			})
			idx++
		}
	}

	files = make([]bytes.Buffer, len(workItems))
	var totalBytes int64

	var wg sync.WaitGroup
	for _, item := range workItems {
		wg.Add(1)
		go func(item workItem) {
			defer wg.Done()
			
			// Create a local font cache for this image generation to ensure thread safety
			// as font.Face is not thread safe
			fontCache := NewFontFaceCache()
			
			pixelMultiplier := getPixelMultiplier(item.imageName, config)
			imageFilename := fmt.Sprintf("%s/%s.jpg", config.HotasImagesDir,
				item.imageName)
			image, err := decodeJpg(imageFilename, log)
			if err != nil || image == nil {
				log.Err("loadImage %s failed. %v", item.imageName, err)
				return
			}
			dc := gg.NewContextForRGBA(image)

			dc.DrawImage(logo, 0, 0)
			xOffset := float64(logo.Bounds().Max.X)
			addImageHeader(dc, &config.ImageHeader, item.profile,
				config.Devices.DeviceLabelsByImage[item.imageName],
				xOffset, pixelMultiplier, config.FontsDir,
				config.InputMinFontSize, fontCache)
			addMRCLogo(dc, &config.Watermark, config.Version, config.Domain,
				xOffset, float64(config.InputPixelXInset), pixelMultiplier,
				config.FontsDir, fontCache)

			// Load the image
			imgBytes := populateImage(dc, imageFilename, image.Bounds().Size(),
				pixelMultiplier, overlaysByProfile[item.profile][item.imageName],
				categories, config, log, fontCache)
			files[item.index] = imgBytes
			atomic.AddInt64(&totalBytes, int64(imgBytes.Len()))
		}(item)
	}
	wg.Wait()
	return files, int(totalBytes)
}

// Returns a sorted list of profile names, a map containing sorted image names
// by profile and a count of files
func prepImgGenData(overlaysByProfile OverlaysByProfile) ([]string,
	map[string][]string, int) {

	profiles := make([]string, 0, len(overlaysByProfile))
	imageNamesByProfile := make(map[string][]string)
	numFiles := 0
	for profile, overlaysByImage := range overlaysByProfile {
		profiles = append(profiles, profile)
		// Generate sorted list of image names
		imageNames := make([]string, 0, len(overlaysByImage))
		for name := range overlaysByImage {
			imageNames = append(imageNames, name)
			numFiles++
		}
		sort.Strings(imageNames)
		imageNamesByProfile[profile] = imageNames
	}
	sort.Strings(profiles)
	return profiles, imageNamesByProfile, numFiles
}

// GenerateImage - generates an image with the provided overlays
func populateImage(dc *gg.Context, imageFilename string, imgSize image.Point,
	pixelMultiplier float64, overlayDataRange map[string]OverlayData,
	categories map[string]string, config *Config, log *Logger,
	fontCache FontLoader) bytes.Buffer {

	width := float64(imgSize.X)
	height := float64(imgSize.Y)

	// Sort keys for deterministic output
	var keys []string
	for k := range overlayDataRange {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		overlayData := overlayDataRange[key]
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
		fontSize = calcFontSize(fullText, fontCache, fontSize, targetWidth,
			targetHeight, config.FontsDir, config.InputFont,
			config.InputMinFontSize)
		// Now create overlays for each text
		// Ugh, second loop through texts
		idx := 0
		for _, context := range prepareContexts(overlayData.ContextToTexts) {
			texts := overlayData.ContextToTexts[context]
			for _, text := range texts {
				largeFont := fontCache.LoadFont(config.FontsDir,
					config.InputFont, fontSize)
				smallFont := fontCache.LoadFont(config.FontsDir,
					config.InputFont, fontSize-1)
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

	var imgBytes bytes.Buffer
	err := jpeg.Encode(&imgBytes, dc.Image(), &jpeg.EncoderOptions{Quality: config.JpgQuality})
	if err != nil {
		log.Err("jpeg encode failed: %v", err)
	}
	return imgBytes
}

func decodeJpg(imageName string, log *Logger) (image *image.RGBA, err error) {
	var r *os.File
	r, err = os.Open(imageName)
	if err != nil {
		log.Err("failed to open: %v", err)
		return
	}
	defer r.Close()

	image, err = jpeg.DecodeIntoRGBA(r, &jpeg.DecoderOptions{})
	if err != nil {
		log.Err("failed to decode: %v", err)
		return
	}
	return
}

func measureString(fontFace font.Face, text string) (int, int) {
	calcX := font.MeasureString(fontFace, text).Round()
	calcY := fontFace.Metrics().Height.Round()
	return calcX, calcY
}

// Resize font till it fits
func calcFontSize(text string, fontLoader FontLoader,
	fontSize int, targetWidth int, targetHeight int, fontsDir string,
	fontName string, minFontSize int) int {
	// Max height in pixels is targetHeight (fontSize = height)
	maxFontSize := targetHeight
	newFontSize := maxFontSize
	for {
		var fontFace font.Face
		if fontLoader == nil {
			fontFace = loadFont(fontsDir, fontName, newFontSize)
		} else {
			fontFace = fontLoader.LoadFont(fontsDir, fontName, newFontSize)
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

func addImageHeader(dc *gg.Context, imageHeader *HeaderData, profile string,
	label string, xOffset float64, pixelMultiplier float64, fontsDir string,
	minFontSize int, fontCache FontLoader) {
	fontSize := int(math.Round(imageHeader.FontSize * pixelMultiplier))
	// Add profile name to header if its not the MRC default
	if profile != ProfileDefault {
		label = fmt.Sprintf("%s (%s)", label, profile)
	}
	targetWidth := dc.Width() -
		int(math.Round(xOffset+2*imageHeader.Inset.X*pixelMultiplier))
	targetHeight := fontSize // Use fontSize as the targetHeight (max height)
	fontSize = calcFontSize(label, fontCache, fontSize, targetWidth, targetHeight,
		fontsDir, imageHeader.Font, minFontSize)
	var headingFont font.Face
	if fontCache != nil {
		headingFont = fontCache.LoadFont(fontsDir, imageHeader.Font, fontSize)
	} else {
		headingFont = loadFont(fontsDir, imageHeader.Font, fontSize)
	}

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

func addMRCLogo(dc *gg.Context, watermark *WatermarkData, version string, domain string,
	xOffset float64, xInset float64, pixelMultiplier float64, fontsDir string,
	fontCache FontLoader) {
	fontSize := int(math.Round(watermark.FontSize * pixelMultiplier))
	// Generate watermark
	text := fmt.Sprintf("%s v%s (%s)", watermark.Text, version, domain)
	var largeFont, smallFont font.Face
	if fontCache != nil {
		largeFont = fontCache.LoadFont(fontsDir, watermark.Font, fontSize)
		smallFont = fontCache.LoadFont(fontsDir, watermark.Font, fontSize-1)
	} else {
		largeFont = loadFont(fontsDir, watermark.Font, fontSize)
		smallFont = loadFont(fontsDir, watermark.Font, fontSize-1)
	}

	drawTextWithBackgroundRec(dc, text, xOffset, watermark.Location, 0, 0,
		fontSize, pixelMultiplier,
		largeFont,
		smallFont,
		watermark.BackgroundColour, watermark.TextColour)
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ankurkotwal/InputRefCard/data"
	"github.com/ankurkotwal/InputRefCard/fs2020"
	"github.com/ankurkotwal/InputRefCard/util"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

var debugOutput bool = false
var verboseOutput bool = false
var configFile = "configs/config.yaml"
var config data.Config
var deviceMap data.DeviceMap
var fontBySize map[int]font.Face

func main() {
	parseCliArgs()

	// Load the configuration
	util.LoadYaml(configFile, &config, debugOutput, "Config")

	// Load the device model (i.e. non-game specific) based on the devices in our game files
	util.LoadYaml(config.DevicesModel, &deviceMap, debugOutput, "Full Device Map")

	// TODO different Font sizes
	fontBySize = make(map[int]font.Face)
	font := util.LoadFont(fmt.Sprintf("%s/%s", config.FontsDir, config.InputFont), float64(config.InputFontSize))
	fontBySize[config.InputFontSize] = font

	fs2020.HandleRequest(generateImage, deviceMap, debugOutput, verboseOutput)
}

func parseCliArgs() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("file\tFlight Simulator 2020 input configration (XML).\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(&debugOutput, "d", false, "Debug output.")
	flag.BoolVar(&verboseOutput, "v", false, "Verbose output.")
	flag.Parse()
	args := flag.Args()
	if len(flag.Args()) < 1 {
		flag.Usage()
		print(args)
		os.Exit(1)
	}

}

func generateImage(overlaysByImage data.OverlaysByImage) {
	for imageFilename, overlayDataRange := range overlaysByImage {
		image, err := gg.LoadImage(fmt.Sprintf("%s/%s", config.ImagesDir, imageFilename))
		if err != nil {
			log.Printf("Error: loadImage %s failed. %v\n", imageFilename, err)
			continue
		}
		dc := gg.NewContextForImage(image)
		dc.SetRGB(0, 0, 0)
		for _, overlayData := range overlayDataRange {
			fontSize := config.InputFontSize
			dc.SetFontFace(fontBySize[fontSize])
			calcX, calcY := dc.MeasureString(overlayData.Text)
			// Resize font till it fits
			for calcX > float64(overlayData.PosAndSize.Width-config.InputPixelInset) ||
				calcY > float64(overlayData.PosAndSize.Height) {
				fontSize -= 2 // Decrement font size
				if font, found := fontBySize[fontSize]; !found {
					font = util.LoadFont(fmt.Sprintf("%s/%s", config.FontsDir, config.InputFont), float64(fontSize))
					fontBySize[fontSize] = font
				}
				dc.SetFontFace(fontBySize[fontSize])
				calcX, calcY = dc.MeasureString(overlayData.Text)
			}
			dc.DrawString(overlayData.Text,
				float64(overlayData.PosAndSize.ImageX+config.InputPixelInset),
				float64(overlayData.PosAndSize.ImageY+config.InputFontSize))
		}
		_ = os.Mkdir("out", os.ModePerm)
		dc.SavePNG(fmt.Sprintf("out/%s", imageFilename))
	}
	// Map the game input bindings to our model
	fmt.Println("Done")
}

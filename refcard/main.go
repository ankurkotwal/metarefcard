package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ankurkotwal/MetaRef/refcard/data"
	"github.com/ankurkotwal/MetaRef/refcard/fs2020"
	"github.com/ankurkotwal/MetaRef/refcard/util"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/font"
)

var configFile = "configs/config.yaml"
var config data.Config
var deviceMap data.DeviceMap

func main() {
	parseCliArgs()

	// Load the configuration
	util.LoadYaml(configFile, &config, "Config")

	// Load the device model (i.e. non-game specific) based on the devices in our game files
	util.LoadYaml(config.DevicesModel, &deviceMap, "Full Device Map")

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.StaticFile("/script.js", "resources/www/script.js")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":   config.AppName,
			"version": config.Version,
		})
	})
	router.POST("/fs2020", func(c *gin.Context) {
		overlaysByImage := fs2020.HandleRequest(deviceMap, config.DebugOutput, config.VerboseOutput)
		files := generateImage(overlaysByImage)
		// TODO Need to return all images
		c.Data(http.StatusOK, "image/jpeg", files[0].Bytes())
	})
	router.Run(":8080")

}

func parseCliArgs() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("file\tFlight Simulator 2020 input configration (XML).\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(flag.Args()) < 1 {
		flag.Usage()
		print(args)
		os.Exit(1)
	}

}

// Return the multiplier/scale of image based on actual width vs default width
func getPixelMultiplier(name string, dc *gg.Context) float64 {
	multiplier := config.PixelMultiplier
	if dimensions, found := config.ImageSizeOverride[name]; found {
		multiplier = float64(dimensions.Width) / float64(config.DefaultImageWidth)
	}
	return multiplier
}

var fontBySize map[float64]font.Face = make(map[float64]font.Face)

func getFontBySize(size float64) font.Face {
	font, found := fontBySize[size]
	if !found {
		name := fmt.Sprintf("%s/%s", config.FontsDir, config.InputFont)
		font = util.LoadFont(name, size)
		fontBySize[size] = font
	}
	return font
}

func generateImage(overlaysByImage data.OverlaysByImage) []*bytes.Buffer {
	var files []*bytes.Buffer = nil
	for imageFilename, overlayDataRange := range overlaysByImage {
		image, err := gg.LoadImage(fmt.Sprintf("%s/%s", config.ImagesDir, imageFilename))
		if err != nil {
			log.Printf("Error: loadImage %s failed. %v\n", imageFilename, err)
			continue
		}
		dc := gg.NewContextForImage(image)
		pixelMultiplier := getPixelMultiplier(imageFilename, dc)
		dc.SetRGB(0, 0, 0)
		for _, overlayData := range overlayDataRange {
			fontSize := float64(config.InputFontSize) * pixelMultiplier
			dc.SetFontFace(getFontBySize(fontSize))
			calcX, calcY := dc.MeasureString(overlayData.Text)
			// Resize font till it fits
			neededWidth := float64(overlayData.PosAndSize.Width-config.InputPixelInset) * pixelMultiplier
			neededHeight := float64(overlayData.PosAndSize.Height) * pixelMultiplier
			for calcX > neededWidth ||
				calcY > neededHeight {
				fontSize -= 2 // Decrement font size
				dc.SetFontFace(getFontBySize(fontSize))
				calcX, calcY = dc.MeasureString(overlayData.Text)
			}
			dc.DrawString(overlayData.Text,
				float64(overlayData.PosAndSize.ImageX+config.InputPixelInset)*pixelMultiplier,
				(float64(overlayData.PosAndSize.ImageY)+config.InputFontSize)*pixelMultiplier)
		}
		var jpgBytes bytes.Buffer
		dc.EncodeJPG(&jpgBytes, &jpeg.Options{Quality: 90})
		files = append(files, &jpgBytes)

		// TODO remove writing to disk or do it only in debug mode
		_ = os.Mkdir("out", os.ModePerm)
		jpgFilename := strings.TrimSuffix(imageFilename, path.Ext(imageFilename)) + ".jpg"
		ioutil.WriteFile(fmt.Sprintf("out/%s", jpgFilename), jpgBytes.Bytes(), 0644)
	}
	// Map the game input bindings to our model
	fmt.Println("Done")
	return files
}

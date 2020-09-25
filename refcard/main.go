package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/ankurkotwal/MetaRef/refcard/data"
	"github.com/ankurkotwal/MetaRef/refcard/fs2020"
	"github.com/ankurkotwal/MetaRef/refcard/util"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/font"
)

type requestHandler func(files [][]byte, deviceMap data.DeviceMap,
	config *data.Config) data.OverlaysByImage

var configFile = "configs/config.yaml"
var config data.Config
var deviceMap data.DeviceMap

func main() {
	exposeGetHandler := false
	parseCliArgs(&exposeGetHandler)

	// Load the configuration
	util.LoadYaml(configFile, &config, "Config")

	// Load the device model (i.e. non-game specific) based on the devices in our game files
	util.LoadYaml(config.DevicesModel, &deviceMap, "Full Device Map")

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.StaticFile("/script.js", "resources/www/script.js")

	// Index page
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":   config.AppName,
			"version": config.Version,
		})
	})

	// Flight simulator endpoint
	router.POST("/fs2020", func(c *gin.Context) {
		// Use the posted form data
		sendResponse(loadFormFiles(c), fs2020.HandleRequest, c)
	})
	if exposeGetHandler {
		router.GET("/fs2020", func(c *gin.Context) {
			// Use local files (specified on the command line)
			sendResponse(loadLocalFiles(), fs2020.HandleRequest, c)
		})
	}

	// Run on port 8080 unless PORT varilable specified
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	router.Run(fmt.Sprintf(":%s", port))

}

func parseCliArgs(exposeGetHandler *bool) {
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("file\tFlight Simulator 2020 input configration (XML).\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(exposeGetHandler, "g", false, "Deploy GET handlers.")
	flag.Parse()
	args := flag.Args()
	if len(flag.Args()) < 1 {
		flag.Usage()
		print(args)
		os.Exit(1)
	}

}

func loadLocalFiles() [][]byte {
	// On the GET route, we'll load our own files (for testing purposes)
	var inputFiles [][]byte
	for _, filename := range flag.Args() {
		file, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Printf("Error reading file. %s\n", err)
		}
		inputFiles = append(inputFiles, file)
	}
	return inputFiles
}

func loadFormFiles(c *gin.Context) [][]byte {
	form, err := c.MultipartForm()
	if err != nil {
		log.Printf("Error getting MultipartForm - %s\n", err)
		return make([][]byte, 0)
	}

	inputFiles := form.File["file"]
	files := make([][]byte, len(inputFiles))
	for idx, file := range inputFiles {
		multipart, err := file.Open()
		if err != nil {
			log.Printf("Error opening multipart file %s - %s\n", file.Filename, err)
			continue
		}
		contents, err := ioutil.ReadAll(multipart)
		if err != nil {
			log.Printf("Error reading multipart file %s - %s\n", file.Filename, err)
			continue
		}
		files[idx] = contents
	}
	return files
}

func sendResponse(loadedFiles [][]byte, handler requestHandler, c *gin.Context) {
	overlaysByImage := handler(loadedFiles, deviceMap, &config)
	generatedFiles := generateImages(overlaysByImage)
	tmplFilename := "templates/refcard.tmpl"
	t, err := template.New(path.Base(tmplFilename)).ParseFiles(tmplFilename)
	if err != nil {
		s := fmt.Sprintf("Error parsing image template - %s\n", err)
		log.Print(s)
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(s))
		return
	}
	imagesAsHTML := []byte{}
	for _, file := range generatedFiles {
		image := struct {
			Base64Contents string
		}{
			Base64Contents: base64.StdEncoding.EncodeToString(file.Bytes()),
		}
		var tpl bytes.Buffer
		if err := t.Execute(&tpl, image); err != nil {
			s := fmt.Sprintf("Error executing image template - %s\n", err)
			log.Print(s)
			continue
		}
		imagesAsHTML = append(imagesAsHTML, tpl.Bytes()...)
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", imagesAsHTML)
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

func generateImages(overlaysByImage data.OverlaysByImage) []*bytes.Buffer {
	var files []*bytes.Buffer = nil
	imageNames := make([]string, 0)
	for name := range overlaysByImage {
		imageNames = append(imageNames, name)
	}
	sort.Strings(imageNames)

	for _, imageFilename := range imageNames {
		overlayDataRange := overlaysByImage[imageFilename]
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

			// Get a list of contexts and sort them
			contexts := make([]string, 0, len(overlayData.ContextToTexts))
			for context := range overlayData.ContextToTexts {
				contexts = append(contexts, context)
			}
			sort.Strings(contexts)
			fullText := ""

			// Iterate through contexts (in order) and texts (already sorted)
			// to generate text to be displayed
			for _, context := range contexts {
				texts := overlayData.ContextToTexts[context]
				for idx, text := range texts {
					padding := "  "
					if idx == 0 {
						padding = ""
					}
					fullText = fmt.Sprintf("%s%s%s", fullText, padding, text)
				}
			}

			calcX, calcY := dc.MeasureString(fullText)
			// Resize font till it fits
			neededWidth := float64(overlayData.PosAndSize.Width-config.InputPixelInset) * pixelMultiplier
			neededHeight := float64(overlayData.PosAndSize.Height) * pixelMultiplier
			for calcX > neededWidth ||
				calcY > neededHeight {
				fontSize-- // Decrement font size
				dc.SetFontFace(getFontBySize(fontSize))
				calcX, calcY = dc.MeasureString(fullText)
			}
			dc.DrawString(fullText,
				float64(overlayData.PosAndSize.ImageX+config.InputPixelInset)*pixelMultiplier,
				(float64(overlayData.PosAndSize.ImageY)+config.InputFontSize)*pixelMultiplier)
		}
		var jpgBytes bytes.Buffer
		dc.EncodeJPG(&jpgBytes, &jpeg.Options{Quality: 90})
		files = append(files, &jpgBytes)
	}
	// Map the game input bindings to our model
	fmt.Println("Done")
	return files
}

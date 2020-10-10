package metarefcard

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

	"github.com/ankurkotwal/MetaRefCard/metarefcard/common"
	"github.com/ankurkotwal/MetaRefCard/metarefcard/fs2020"
	"github.com/ankurkotwal/MetaRefCard/metarefcard/sws"
	"github.com/fogleman/gg"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/font"
)

type requestHandler func(files [][]byte,
	config *common.Config) (common.OverlaysByImage, map[string]string)

var config common.Config
var exposeGetHandler = false

// Initialise the package
func initialise() {
	parseCliArgs(&exposeGetHandler)

	// Load the configuration
	common.LoadYaml("config/config.yaml", &config, "Config")

	// Load the device model (i.e. non-game specific)
	var generatedConfig common.GeneratedConfig
	common.LoadYaml(config.DevicesFile, &generatedConfig, "Full Device Map")

	// Add device additions to the main device index
	for shortName, inputs := range config.DeviceMap {
		generatedInputs, found := generatedConfig.DeviceMap[shortName]
		if !found {
			generatedInputs = make(common.DeviceInputs)
			generatedConfig.DeviceMap[shortName] = generatedInputs
		}

		// Already have some inputs. Need to override one at a time
		for input, additionalInput := range inputs {
			generatedInputs[input] = additionalInput
		}
	}
	config.DeviceMap = generatedConfig.DeviceMap

	// Add input overrides
	for shortName, inputOverrides := range config.InputOverrides {
		deviceInputs, found := config.DeviceMap[shortName]
		if !found {
			log.Printf("Error: Override device not found %s\n", shortName)
			continue // Next device
		}
		for input, override := range inputOverrides {
			deviceInputs[input] = override
		}
	}

	// Add image map additions
	for shortName, image := range config.ImageMap {
		generatedConfig.ImageMap[shortName] = image
	}
	config.ImageMap = generatedConfig.ImageMap

}

// RunLocal will run local files
func RunLocal(files []string) {
	initialise()
	sendResponse(loadLocalFiles(files), fs2020.HandleRequest, nil)
}

// RunServer will run the server
func RunServer() {
	initialise()

	router := gin.Default()
	router.LoadHTMLGlob("resources/web_templates/*")
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
			sendResponse(loadLocalFiles(flag.Args()), fs2020.HandleRequest, c)
		})
	}

	// Flight simulator endpoint
	router.POST("/sws", func(c *gin.Context) {
		// Use the posted form data
		sendResponse(loadFormFiles(c), sws.HandleRequest, c)
	})
	if exposeGetHandler {
		router.GET("/sws", func(c *gin.Context) {
			// Use local files (specified on the command line)
			sendResponse(loadLocalFiles(flag.Args()), sws.HandleRequest, c)
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
		fmt.Printf("file\tSupported game input configration.\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(exposeGetHandler, "g", false, "Deploy GET handlers.")
	// TODO add files list for --fs2020 and --sws
	flag.Parse()
}

func loadLocalFiles(files []string) [][]byte {
	// On the GET route, we'll load our own files (for testing purposes)
	var inputFiles [][]byte
	for _, filename := range files {
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
	// Call game handler to generate image overlayes
	overlaysByImage, categories := handler(loadedFiles, &config)

	// Now generate images from the overlays
	generatedFiles := generateImages(overlaysByImage, categories)

	// Generate HTML
	tmplFilename := "resources/web_templates/refcard.tmpl"
	t, err := template.New(path.Base(tmplFilename)).ParseFiles(tmplFilename)
	if err != nil {
		s := fmt.Sprintf("Error parsing image template - %s\n", err)
		log.Print(s)
		if c != nil {
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(s))
		}
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
	if c != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", imagesAsHTML)
	}
}

// Return the multiplier/scale of image based on actual width vs default width
func getPixelMultiplier(name string, dc *gg.Context) float64 {
	multiplier := config.PixelMultiplier
	if dimensions, found := config.ImageSizeOverride[name]; found {
		multiplier = float64(dimensions.W) / float64(config.DefaultImage.W)
	}
	return multiplier
}

var fontBySize map[float64]font.Face = make(map[float64]font.Face)

func getFontBySize(size float64) font.Face {
	font, found := fontBySize[size]
	if !found {
		name := fmt.Sprintf("%s/%s", config.FontsDir, config.InputFont)
		font = common.LoadFont(name, size)
		fontBySize[size] = font
	}
	return font
}

func prepareGeneratorData(overlaysByImage common.OverlaysByImage) []string {
	// Generate sorted list of image names
	imageNames := make([]string, 0)
	for name := range overlaysByImage {
		imageNames = append(imageNames, name)
	}
	sort.Strings(imageNames)
	return imageNames
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

func generateImages(overlaysByImage common.OverlaysByImage, categories map[string]string) []*bytes.Buffer {
	var files []*bytes.Buffer = nil

	imageNames := prepareGeneratorData(overlaysByImage)

	for _, imageFilename := range imageNames {
		image, err := gg.LoadImage(fmt.Sprintf("%s/%s.png", config.ImagesDir, imageFilename))
		if err != nil {
			log.Printf("Error: loadImage %s failed. %v\n", imageFilename, err)
			continue
		}

		// Load the image
		dc := gg.NewContext(image.Bounds().Size().X, image.Bounds().Size().Y)
		// Set the background colour
		dc.SetHexColor(config.BackgroundColour)
		dc.Clear()
		// Apply the image on top
		dc.DrawImage(image, 0, 0)
		dc.SetRGB(0, 0, 0)
		pixelMultiplier := getPixelMultiplier(imageFilename, dc)

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

			fontSize := float64(config.InputFontSize) * pixelMultiplier
			dc.SetFontFace(getFontBySize(fontSize))

			targetWidth := float64(overlayData.PosAndSize.W-config.InputPixelInset) * pixelMultiplier
			targetHeight := float64(overlayData.PosAndSize.H) * pixelMultiplier

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
			fontSize = calcFontSize(fullText, fontSize, targetWidth, targetHeight)
			// Now create overlays for each text
			// Ugh, second loop through texts
			idx := 0
			for _, context := range prepareContexts(overlayData.ContextToTexts) {
				texts := overlayData.ContextToTexts[context]
				for _, text := range texts {
					dc.SetFontFace(getFontBySize(fontSize))
					offset, _ := dc.MeasureString(incrementalTexts[idx])
					idx++

					x := offset + float64(overlayData.PosAndSize.X+config.InputPixelInset)*pixelMultiplier
					y := float64(overlayData.PosAndSize.Y) * pixelMultiplier
					w, h := dc.MeasureString(text)
					// Vertically center
					y = y + (targetHeight-h)/2

					dc.SetHexColor(categories[context])
					dc.DrawRoundedRectangle(x, y, w, h, 6)
					dc.Fill()
					dc.SetHexColor(config.LightColour)
					dc.SetFontFace(getFontBySize(fontSize - 1)) // Render one font size smaller to fit in rect
					w2, _ := dc.MeasureString(text)
					dc.DrawStringAnchored(text, x+(w-w2)/2, y, 0, 0.85)
				}
			}
		}
		var jpgBytes bytes.Buffer
		dc.EncodeJPG(&jpgBytes, &jpeg.Options{Quality: 90})
		files = append(files, &jpgBytes)
	}
	return files
}

var fontCtx *gg.Context = nil

// Resize font till it fits
func calcFontSize(text string, fontSize float64, targetWidth float64, targetHeight float64) float64 {
	if fontCtx == nil {
		fontCtx = gg.NewContext(1, 1) // Only needed for fonts, small size
	}
	fontCtx.SetFontFace(getFontBySize(fontSize))
	calcX, calcY := fontCtx.MeasureString(text)
	if calcX > targetWidth || calcY > targetHeight {
		// Text is too big, shrink till it fits
		for calcX > targetWidth || calcY > targetHeight {
			fontSize--
			fontCtx.SetFontFace(getFontBySize(fontSize))
			calcX, calcY = fontCtx.MeasureString(text)
		}
	} else if calcX < targetWidth && calcY < targetHeight {
		// Text can grow to fit
		for calcX < targetWidth && calcY < targetHeight {
			fontSize++
			fontCtx.SetFontFace(getFontBySize(fontSize))
			calcX, calcY = fontCtx.MeasureString(text)
		}
		fontSize-- // Go down one size
	}
	return fontSize
}

package metarefcard

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"image"
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
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

// requestHandler - handles incoming requests and returns game data, game binds,
// neededDevices and a context to colour mapping
type requestHandler func(files [][]byte, config *common.Config) (*common.GameData,
	common.GameBindsByDevice, common.MockSet, common.MockSet, string)

var config common.Config
var debugMode = false

// Initialise the package
func initialise() gameFiles {
	gameFiles := parseCliArgs(&debugMode)

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

	return gameFiles
}

// RunLocal will run local files
func RunLocal() {
	gameFiles := initialise()
	sendResponse(loadLocalFiles(gameFiles.fs2020), fs2020.HandleRequest,
		fs2020.MatchGameInputToModel, nil)
}

// RunServer will run the server
func RunServer() {
	gameFiles := initialise()

	router := gin.Default()

	if debugMode {
		pprof.Register(router)
	}

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
		sendResponse(loadFormFiles(c), fs2020.HandleRequest,
			fs2020.MatchGameInputToModel, c)
	})
	if debugMode {
		router.GET("/fs2020", func(c *gin.Context) {
			// Use local files (specified on the command line)
			sendResponse(loadLocalFiles(gameFiles.fs2020), fs2020.HandleRequest,
				fs2020.MatchGameInputToModel, c)
		})
	}

	// Flight simulator endpoint
	router.POST("/sws", func(c *gin.Context) {
		// Use the posted form data
		sendResponse(loadFormFiles(c), sws.HandleRequest,
			sws.MatchGameInputToModel, c)
	})
	if debugMode {
		router.GET("/sws", func(c *gin.Context) {
			// Use local files (specified on the command line)
			sendResponse(loadLocalFiles(gameFiles.sws), sws.HandleRequest,
				sws.MatchGameInputToModel, c)
		})
	}

	// Run on port 8080 unless PORT varilable specified
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	router.Run(fmt.Sprintf(":%s", port))

}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type gameFiles struct {
	fs2020 arrayFlags
	sws    arrayFlags
}

func parseCliArgs(debugMode *bool) gameFiles {
	var gameFiles gameFiles
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("file\tSupported game input configration.\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(debugMode, "d", false, "Debug mode & deploy GET handlers.")
	flag.Var(&gameFiles.fs2020, "fs2020", "Flight Simulator 2020 input configs")
	flag.Var(&gameFiles.sws, "sws", "Star Wars Squadrons input configs")
	flag.Parse()

	return gameFiles
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

func sendResponse(loadedFiles [][]byte, handler requestHandler,
	matchFunc common.MatchGameInputToModel, c *gin.Context) {
	// Call game handler to generate image overlayes
	gameData, gameBinds, gameDevices, gameContexts, gameLabel := handler(loadedFiles, &config)
	overlaysByImage := common.PopulateImageOverlays(gameDevices, &config,
		gameBinds, gameData, matchFunc)

	// Now generate images from the overlays
	generatedFiles := generateImages(overlaysByImage, gameContexts, gameLabel)

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

func prepareGeneratorData(overlaysByImage common.OverlaysByImage) []string {
	// Generate sorted list of image names
	imageNames := make([]string, 0)
	for name := range overlaysByImage {
		imageNames = append(imageNames, name)
	}
	sort.Strings(imageNames)
	return imageNames
}

type chanData struct {
	Dc            *gg.Context
	Image         *image.Image
	ImageFilename string
}

func generateImages(overlaysByImage common.OverlaysByImage, categories map[string]string,
	gameLabel string) []*bytes.Buffer {

	imageNames := prepareGeneratorData(overlaysByImage)

	files := make([]*bytes.Buffer, 0, len(imageNames))
	channel := make(chan chanData, len(imageNames))
	for _, imageName := range imageNames {
		go func(imageFilename string) {
			image, err := gg.LoadImage(fmt.Sprintf("%s/%s.png",
				config.HotasImagesDir, imageFilename))
			if err != nil {
				log.Printf("Error: loadImage %s failed. %v\n", imageFilename, err)
				channel <- chanData{nil, nil, ""}
			}

			// Load the image
			dc := gg.NewContext(image.Bounds().Size().X, image.Bounds().Size().Y)

			channel <- chanData{dc, &image, imageFilename}
		}(imageName)

	}

	imagesByName := make(map[string]*chanData)
	for range imageNames {
		data := <-channel
		imagesByName[data.ImageFilename] = &data
	}

	for _, imageFilename := range imageNames {
		data := imagesByName[imageFilename]
		// Sort by image filename
		imgBytes := common.GenerateImage(data.Dc, data.Image, data.ImageFilename,
			overlaysByImage, categories, &config, gameLabel)
		if imgBytes != nil {
			files = append(files, imgBytes)
		}
	}
	return files
}

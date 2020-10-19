package metarefcard

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"image"
	"io/ioutil"
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

var config *common.Config
var debugMode = false
var logs []*common.LogEntry = make([]*common.LogEntry, 0)

type gameInfo func() (string, string, common.FuncRequestHandler,
	common.FuncMatchGameInputToModel)

// Initialise the package
func initialise() (cliGameArgs, []gameInfo) {
	gamesInfo := []gameInfo{fs2020.GetGameInfo, sws.GetGameInfo}

	// Capture logs
	common.RegisterHandler(func(newLog *common.LogEntry) {
		logs = append(logs, newLog)
	})
	gameArgs := parseCliArgs(&debugMode, gamesInfo)

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
			common.LogErr("Override device not found %s", shortName)
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

	return gameArgs, gamesInfo
}

// RunServer will run the server
func RunServer() {
	gameArgs, gamesInfo := initialise()

	router := gin.Default()
	if debugMode {
		pprof.Register(router)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router.LoadHTMLGlob("resources/www/templates/*.html")
	router.StaticFile("/favicon.ico", "resources/www/static/favicon.ico")
	router.StaticFile("/main.css", "resources/www/static/main.css")
	router.StaticFile("/script.js", "resources/www/static/script.js")

	// Index page
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/fs2020")
	})

	for _, game := range gamesInfo {
		label, _, handleRequest, matchGameInputToModel := game()
		router.GET(fmt.Sprintf("/%s", label), func(c *gin.Context) {
			c.HTML(http.StatusOK, fmt.Sprintf("%s.html", label), gin.H{
				"Title":   config.AppName,
				"Version": config.Version,
			})
		})
		// Flight simulator endpoint
		router.POST(fmt.Sprintf("/api/%s", label), func(c *gin.Context) {
			// Use the posted form data
			sendResponse(loadFormFiles(c), handleRequest, matchGameInputToModel, c)
		})
		if debugMode {
			router.GET(fmt.Sprintf("/test/%s", label), func(c *gin.Context) {
				// Use local files (specified on the command line)
				sendResponse(loadLocalFiles(*gameArgs[label]), handleRequest,
					matchGameInputToModel, c)
			})
		}

	}

	// Run on port 8080 unless PORT varilable specified
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	router.Run(fmt.Sprintf(":%s", port))

}

type cliGameArgs map[string]*arrayFlags

// arrayFlags are used for storing a list of CLI values
type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

// Set adds to the ArrayFlag
func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func parseCliArgs(debugMode *bool, games []gameInfo) cliGameArgs {
	gameFiles := make(cliGameArgs)
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("file\tSupported game input configration.\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(debugMode, "d", false, "Debug mode & deploy GET handlers.")
	for _, getGameInfo := range games {
		label, desc, _, _ := getGameInfo()
		args, found := gameFiles[label]
		if !found {
			arrayFlags := make(arrayFlags, 0)
			args = &arrayFlags
			gameFiles[label] = args
		}
		flag.Var(args, label, desc)
	}
	flag.Parse()

	return gameFiles
}

func loadLocalFiles(files []string) [][]byte {
	// On the GET route, we'll load our own files (for testing purposes)
	var inputFiles [][]byte
	for _, filename := range files {
		file, err := ioutil.ReadFile(filename)
		if err != nil {
			common.LogErr("Error reading file. %s", err)
		}
		inputFiles = append(inputFiles, file)
	}
	return inputFiles
}

func loadFormFiles(c *gin.Context) [][]byte {
	form, err := c.MultipartForm()
	if err != nil {
		common.LogErr("Error getting MultipartForm - %s", err)
		return make([][]byte, 0)
	}

	inputFiles := form.File["file"]
	files := make([][]byte, len(inputFiles))
	for idx, file := range inputFiles {
		multipart, err := file.Open()
		if err != nil {
			common.LogErr("Error opening multipart file %s - %s", file.Filename, err)
			continue
		}
		contents, err := ioutil.ReadAll(multipart)
		if err != nil {
			common.LogErr("Error reading multipart file %s - %s", file.Filename, err)
			continue
		}
		files[idx] = contents
	}
	return files
}

func sendResponse(loadedFiles [][]byte, handler common.FuncRequestHandler,
	matchFunc common.FuncMatchGameInputToModel, c *gin.Context) {
	// Call game handler to generate image overlayes
	gameData, gameBinds, gameDevices, gameContexts, gameLogo :=
		handler(loadedFiles, config)
	overlaysByImage := common.PopulateImageOverlays(gameDevices, config,
		gameBinds, gameData, matchFunc)

	// Now generate images from the overlays
	generatedFiles := generateImages(overlaysByImage, gameContexts, gameLogo)

	// Generate HTML
	cardTempl := "resources/www/templates/refcard.html"
	t, err := template.New(path.Base(cardTempl)).ParseFiles(cardTempl)
	if err != nil {
		s := fmt.Sprintf("Error parsing card template - %s", err)
		common.LogErr(s)
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
			common.LogErr(fmt.Sprintf("Error executing image template - %s", err))
			continue
		}
		imagesAsHTML = append(imagesAsHTML, tpl.Bytes()...)
	}
	// Generate HTML
	logTempl := "resources/www/templates/log.html"
	l, err := template.New(path.Base(logTempl)).ParseFiles(logTempl)
	if err != nil {
		s := fmt.Sprintf("Error parsing logging template - %s", err)
		common.LogErr(s)
		if c != nil {
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(s))
		}
	} else {
		var tpl bytes.Buffer
		err = l.Execute(&tpl, struct{ Logs []*common.LogEntry }{Logs: logs})
		if err != nil {
			common.LogErr("Error executing logging template - %s", err)
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
	channel := make(chan chanData)
	for _, imageName := range imageNames {
		go func(imageFilename string) {
			image, err := gg.LoadImage(fmt.Sprintf("%s/%s.png",
				config.HotasImagesDir, imageFilename))
			if err != nil {
				common.LogErr("loadImage %s failed. %v", imageFilename, err)
				channel <- chanData{nil, nil, ""}
			}

			// Load the image
			dc := gg.NewContext(image.Bounds().Size().X, image.Bounds().Size().Y)

			channel <- chanData{dc, &image, imageFilename}
		}(imageName)

	}

	imagesByName := make(map[string]*chanData)
	for i := 0; i < len(imageNames); i++ {
		data := <-channel
		imagesByName[data.ImageFilename] = &data
	}

	for _, imageFilename := range imageNames {
		data := imagesByName[imageFilename]
		// Sort by image filename
		imgBytes := common.GenerateImage(data.Dc, data.Image, data.ImageFilename,
			overlaysByImage, categories, config, gameLabel)
		if imgBytes != nil {
			files = append(files, imgBytes)
		}
	}
	return files
}

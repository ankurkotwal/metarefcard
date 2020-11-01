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

type gameInfo func() (string, string, common.FuncRequestHandler,
	common.FuncMatchGameInputToModel)

// Initialise the package
func initialise(log *common.Logger) (cliGameArgs, []gameInfo) {
	gamesInfo := []gameInfo{fs2020.GetGameInfo, sws.GetGameInfo}
	gameArgs := parseCliArgs(&debugMode, gamesInfo)

	// Load the configuration
	common.LoadYaml("config/config.yaml", &config, "Config", log)

	// Load the device information
	common.LoadDevicesInfo(config.DevicesFile, &config.Devices, log)

	return gameArgs, gamesInfo
}

// RunServer will run the server
func RunServer() {
	log := common.NewLog()
	gameArgs, gamesInfo := initialise(log)

	router := gin.Default()
	if debugMode {
		pprof.Register(router)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router.LoadHTMLGlob("resources/www/templates/*.html")
	router.StaticFile("/favicon.ico", "resources/www/static/favicon.ico")
	router.StaticFile("/logo_metarefcard.png", "resources/www/static/logo_metarefcard.png")
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
			sendResponse(loadFormFiles(c, log), handleRequest, matchGameInputToModel, c)
		})
		if debugMode {
			router.GET(fmt.Sprintf("/test/%s", label), func(c *gin.Context) {
				// Use local files (specified on the command line)
				sendResponse(loadLocalFiles(*gameArgs[label], log), handleRequest,
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

func loadLocalFiles(files []string, log *common.Logger) [][]byte {
	var inputFiles [][]byte
	for _, filename := range files {
		file, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Err("Error reading file. %s", err)
		}
		inputFiles = append(inputFiles, file)
	}
	return inputFiles
}

func loadFormFiles(c *gin.Context, log *common.Logger) [][]byte {
	form, err := c.MultipartForm()
	if err != nil {
		log.Err("Error getting MultipartForm - %s", err)
		return make([][]byte, 0)
	}

	inputFiles := form.File["file"]
	files := make([][]byte, len(inputFiles))
	for idx, file := range inputFiles {
		multipart, err := file.Open()
		if err != nil {
			log.Err("Error opening multipart file %s - %s", file.Filename, err)
			continue
		}
		contents, err := ioutil.ReadAll(multipart)
		if err != nil {
			log.Err("Error reading multipart file %s - %s", file.Filename, err)
			continue
		}
		files[idx] = contents
	}
	return files
}

func sendResponse(loadedFiles [][]byte, handler common.FuncRequestHandler,
	matchFunc common.FuncMatchGameInputToModel, c *gin.Context) {
	log := common.NewLog()

	// Call game handler to generate image overlayes
	gameData, gameBinds, gameDevices, gameContexts, gameLogo :=
		handler(loadedFiles, config, log)
	overlaysByImage := common.PopulateImageOverlays(gameDevices, config, log,
		gameBinds, gameData, matchFunc)

	// Now generate images from the overlays
	generatedFiles := generateImages(overlaysByImage, gameContexts, gameLogo, log)

	// Generate HTML
	cardTempl := "resources/www/templates/refcard.html"
	t, err := template.New(path.Base(cardTempl)).ParseFiles(cardTempl)
	if err != nil {
		s := fmt.Sprintf("Error parsing card template - %s", err)
		log.Err(s)
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
			log.Err(fmt.Sprintf("Error executing image template - %s", err))
			continue
		}
		imagesAsHTML = append(imagesAsHTML, tpl.Bytes()...)
	}
	// Generate HTML
	logTempl := "resources/www/templates/log.html"
	l, err := template.New(path.Base(logTempl)).ParseFiles(logTempl)
	if err != nil {
		s := fmt.Sprintf("Error parsing logging template - %s", err)
		log.Err(s)
		if c != nil {
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(s))
		}
	} else {
		var tpl bytes.Buffer
		err = l.Execute(&tpl, struct{ Logs []*common.LogEntry }{Logs: *log})
		if err != nil {
			log.Err("Error executing logging template - %s", err)
		}
		imagesAsHTML = append(imagesAsHTML, tpl.Bytes()...)
	}
	if c != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", imagesAsHTML)
	}
}

// Returns a sorted list of profile names, a map containing sorted image names by profile and a count of files
func prepareGeneratorData(overlaysByProfile common.OverlaysByProfile) ([]string, map[string][]string, int) {
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

type chanData struct {
	Dc            *gg.Context
	Profile       string
	Image         *image.Image
	ImageFilename string
}

func generateImages(overlaysByProfile common.OverlaysByProfile, categories map[string]string,
	gameLabel string, log *common.Logger) []*bytes.Buffer {

	profiles, imageNamesByProfile, numFiles := prepareGeneratorData(overlaysByProfile)
	channel := make(chan chanData)

	for profile, imagesNames := range imageNamesByProfile {
		for _, imageName := range imagesNames {
			go func(imageFilename string, profileName string) {
				image, err := gg.LoadImage(fmt.Sprintf("%s/%s.png",
					config.HotasImagesDir, imageFilename))
				if err != nil {
					log.Err("loadImage %s failed. %v", imageFilename, err)
					channel <- chanData{nil, "", nil, ""}
				}

				// Load the image
				dc := gg.NewContext(image.Bounds().Size().X, image.Bounds().Size().Y)

				channel <- chanData{dc, profileName, &image, imageFilename}
			}(imageName, profile)

		}
	}

	// Create a map of images to generate because we need to create them in
	// a sorted order.
	imagesToGenerate := make(map[string]map[string]*chanData)
	for i := 0; i < numFiles; i++ {
		data := <-channel
		imageByName, found := imagesToGenerate[data.Profile]
		if !found {
			// New profile, add to index
			imageByName = make(map[string]*chanData)
			imagesToGenerate[data.Profile] = imageByName
		}
		imageByName[data.ImageFilename] = &data
	}

	files := make([]*bytes.Buffer, 0, numFiles)
	// Iterate using the sorted structures to have order stability
	for _, profile := range profiles {
		for _, imageName := range imageNamesByProfile[profile] {
			data := imagesToGenerate[profile][imageName]
			imgBytes := common.GenerateImage(data.Dc, data.Image, data.ImageFilename,
				profile, overlaysByProfile[profile], categories, config, log, gameLabel)
			if imgBytes != nil {
				files = append(files, imgBytes)
			}
		}
	}
	return files
}

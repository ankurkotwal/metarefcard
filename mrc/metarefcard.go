package mrc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/ankurkotwal/metarefcard/mrc/common"
	"github.com/ankurkotwal/metarefcard/mrc/fs2020"
	"github.com/ankurkotwal/metarefcard/mrc/sws"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

var config *common.Config

// GameInfo is the info needed to fit into MetaRefCard
// Returns:
//   * Game label / name
//   * User friendly command line description
//   * Func handler for incoming request
//   * Func that matches the game input format to MRC's model
type GameInfo func() (string, string, common.FuncRequestHandler,
	common.FuncMatchGameInputToModel)

// GamesInfo returns GameInfo
var GamesInfo []GameInfo = []GameInfo{fs2020.GetGameInfo, sws.GetGameInfo}

// GetServer will run the server
func GetServer(debugMode bool, gameArgs GameToInputFiles) (*gin.Engine, string) {
	log := common.NewLog()
	// Load the configuration
	common.LoadYaml("config/config.yaml", &config, "Config", log)
	// Load the device information
	common.LoadDevicesInfo(config.DevicesFile, &config.Devices, log)

	if !debugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	if debugMode {
		pprof.Register(router)
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

	for _, game := range GamesInfo {
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
	return router, fmt.Sprintf(":%s", port)
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
	generatedFiles, _ := common.GenerateImages(overlaysByImage, gameContexts,
		gameLogo, config, log)

	// Generate HTML
	cardTempl := "resources/www/templates/refcard.html"
	t, err := template.New(path.Base(cardTempl)).ParseFiles(cardTempl)
	if err != nil {
		s := fmt.Sprintf("Error parsing card template - %s", err)
		log.Err(s)
		if c != nil {
			c.Data(http.StatusInternalServerError, "text/html; charset=utf-8",
				[]byte(s))
		}
		return
	}

	type base64Image struct {
		Base64Contents string
	}
	for _, file := range generatedFiles {
		image := base64Image{
			Base64Contents: base64.StdEncoding.EncodeToString(file.Bytes()),
		}
		var tpl bytes.Buffer
		if err := t.Execute(&tpl, image); err != nil {
			log.Err(fmt.Sprintf("Error executing image template - %s", err))
			continue
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
	}
	// Generate HTML
	logTempl := "resources/www/templates/log.html"
	l, err := template.New(path.Base(logTempl)).ParseFiles(logTempl)
	if err != nil {
		s := fmt.Sprintf("Error parsing logging template - %s", err)
		log.Err(s)
		c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(s))
	} else {
		var tpl bytes.Buffer
		err = l.Execute(&tpl, struct{ Logs []*common.LogEntry }{Logs: *log})
		if err != nil {
			log.Err("Error executing logging template - %s", err)
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
	}
}

// GameToInputFiles are the per-game arguments specified on the command line
type GameToInputFiles map[string]*Filenames

// Filenames are used for storing a list of CLI values
type Filenames []string

func (i *Filenames) String() string {
	return ""
}

// Set adds to the ArrayFlag
func (i *Filenames) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// GetFilesFromDir returns a list of file names from a directory
func GetFilesFromDir(path string) *Filenames {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	testFiles := make(Filenames, 0, len(files))
	for _, f := range files {
		if !f.IsDir() {
			testFiles = append(testFiles, fmt.Sprintf("%s/%s", path, f.Name()))
		}
	}
	return &testFiles
}

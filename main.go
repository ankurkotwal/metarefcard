package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/ankurkotwal/metarefcard/mrc"
)

func main() {
	if err := runServer(defaultRunner); err != nil {
		log.Fatal(err)
	}
}

// ServerRunner is a function type that runs a gin.Engine on a port
type ServerRunner func(*gin.Engine, string) error

// defaultRunner is the default server runner that starts the HTTP server
func defaultRunner(router *gin.Engine, port string) error {
	return router.Run(port)
}

// runServer contains the main application logic and is extracted for testability.
// The runner parameter allows injecting a mock for testing.
func runServer(runner ServerRunner) error {
	debugMode, gameArgs := parseCliArgs()
	router, port := mrc.GetServer(debugMode, gameArgs)
	return runner(router, port)
}

func parseCliArgs() (bool, mrc.GameToInputFiles) {
	gameFiles := make(mrc.GameToInputFiles)
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("file\tSupported game input configration.\n")
		flag.PrintDefaults()
	}
	var debugMode bool
	flag.BoolVar(&debugMode, "d", false, "Enable debug mode & deploy GET handlers.")
	var testDataDir string
	flag.StringVar(&testDataDir, "t", "", "Directory to load test data from. Only used if debug mode is enabled.")
	flag.Parse()
	// If in debug mode and a test data dir was provided, read files by game label dir
	if debugMode && len(testDataDir) > 0 {
		for _, getGameInfo := range mrc.GamesInfo {
			label, _, _, _ := getGameInfo()
			files, err := mrc.GetFilesFromDir(fmt.Sprintf("%s/%s", testDataDir, label))
			if err != nil {
				log.Printf("Error loading files for %s: %v", label, err)
				continue
			}
			gameFiles[label] = files
		}
	}

	return debugMode, gameFiles
}

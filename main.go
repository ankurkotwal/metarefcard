package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ankurkotwal/metarefcard/mrc"
)

func main() {
	debugMode, gameArgs := parseCliArgs()
	router, port := mrc.GetServer(debugMode, gameArgs)
	err := router.Run(port)
	if err != nil {
		log.Fatal(err)
	}
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

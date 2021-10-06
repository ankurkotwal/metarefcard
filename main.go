package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ankurkotwal/MetaRefCard/metarefcard"
)

func main() {
	debugMode, gameArgs := parseCliArgs()
	router, port := metarefcard.GetServer(debugMode, gameArgs)
	err := router.Run(port)
	if err != nil {
		log.Fatal(err)
	}
}

func parseCliArgs() (bool, metarefcard.GameToInputFiles) {
	gameFiles := make(metarefcard.GameToInputFiles)
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
		for _, getGameInfo := range metarefcard.GamesInfo {
			label, _, _, _ := getGameInfo()
			gameFiles[label] = metarefcard.GetFilesFromDir(fmt.Sprintf("%s/%s", testDataDir, label))
		}
	}

	return debugMode, gameFiles
}

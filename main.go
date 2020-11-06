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

func parseCliArgs() (bool, metarefcard.CliGameArgs) {
	gameFiles := make(metarefcard.CliGameArgs)
	var debugMode bool
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("file\tSupported game input configration.\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(&debugMode, "d", false, "Debug mode & deploy GET handlers.")
	for _, getGameInfo := range metarefcard.GamesInfo {
		label, desc, _, _ := getGameInfo()
		args, found := gameFiles[label]
		if !found {
			arrayFlags := make(metarefcard.ArrayFlags, 0)
			args = &arrayFlags
			gameFiles[label] = args
		}
		flag.Var(args, label, desc)
	}
	flag.Parse()

	return debugMode, gameFiles
}

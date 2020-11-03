package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ankurkotwal/MetaRefCard/metarefcard"
)

func BenchmarkFS2020(b *testing.B) {
	router, _ := metarefcard.GetServer(true, getTestGameArgs())

	for n := 0; n < b.N; n++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/sws", nil)
		router.ServeHTTP(w, req)
	}
}

type filesByGame map[string][]string

func getTestGameArgs() metarefcard.CliGameArgs {
	cliGameArgs := make(metarefcard.CliGameArgs)
	cliGameArgs["fs2020"] = getFilesFromDir("testdata/fs2020")
	cliGameArgs["sws"] = getFilesFromDir("testdata/sws")
	return cliGameArgs
}

func getFilesFromDir(path string) *metarefcard.ArrayFlags {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	testFiles := make(metarefcard.ArrayFlags, 0, len(files))
	for _, f := range files {
		if !f.IsDir() {
			testFiles = append(testFiles, fmt.Sprintf("%s/%s", path, f.Name()))
		}
	}
	return &testFiles
}

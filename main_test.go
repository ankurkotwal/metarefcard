package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ankurkotwal/MetaRefCard/metarefcard"
)

func TestSwsSerial(t *testing.T) {
	runTestSerial(t, "/test/sws", 50)
}

func TestSwsConc(t *testing.T) {
	runTestConc(t, "/test/sws", 50)
}

func TestFs2020Serial(t *testing.T) {
	runTestSerial(t, "/test/fs2020", 25)
}

func TestFs2020Conc(t *testing.T) {
	runTestConc(t, "/test/fs2020", 25)
}

func runTestSerial(t *testing.T, url string, N int) {
	router, _ := metarefcard.GetServer(true, getTestGameArgs())
	time.Sleep(2 * time.Second) // a bit of time to properly init...

	for n := 0; n < N; n++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", url, nil)
		router.ServeHTTP(w, req)
	}
	// exit unconditionnaly after the requests
}

func runTestConc(t *testing.T, url string, N int) {
	router, _ := metarefcard.GetServer(true, getTestGameArgs())
	time.Sleep(2 * time.Second) // a bit of time to properly init...

	var wg sync.WaitGroup
	wg.Add(N)
	for n := 0; n < N; n++ {
		time.Sleep(100 * time.Millisecond)
		go func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", url, nil)
			router.ServeHTTP(w, req)
			wg.Done()
		}()
	}
	wg.Wait()

	// exit unconditionnaly after the requests
}

func getTestGameArgs() metarefcard.GameToInputFiles {
	cliGameArgs := make(metarefcard.GameToInputFiles)
	cliGameArgs["fs2020"] = metarefcard.GetFilesFromDir("testdata/fs2020")
	cliGameArgs["sws"] = metarefcard.GetFilesFromDir("testdata/sws")
	return cliGameArgs
}

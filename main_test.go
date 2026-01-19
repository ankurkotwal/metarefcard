package main

import (
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ankurkotwal/metarefcard/mrc"
)

func TestParseCliArgs(t *testing.T) {
	// Save original args
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Test case 1: Debug mode
	os.Args = []string{"cmd", "-d", "-t", "testdata"}
	debug, _ := parseCliArgs() // gameArgs might differ based on testdata existence
	if !debug {
		t.Error("Expected debug mode true")
	}

	// Test case 2: Default
	os.Args = []string{"cmd"}
	// Reset flags because they stick around in global state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	
	debug, _ = parseCliArgs()
	if debug {
		t.Error("Expected debug mode false")
	}
	
	// Cover usage function
	// We need to ensure flag.Usage is set first, which parseCliArgs does.
	// Calling parseCliArgs sets flag.Usage.
	if flag.Usage != nil {
		flag.Usage()
	}
}

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

func TestResponseContent(t *testing.T) {
	router, _ := mrc.GetServer(true, getTestGameArgs())

	tests := []struct {
		name     string
		url      string
		contains []string
	}{
		{"FS2020 UI", "/test/fs2020", []string{"<hr class=\"my-4 solid\">", "data:image/jpg;base64"}},
		{"SWS UI", "/test/sws", []string{"<hr class=\"my-4 solid\">"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tt.url, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status OK, got %v", w.Code)
			}
			respBody := w.Body.String()

			// Debug: print body length and first 100 chars
			t.Logf("Response Body Length: %d", len(respBody))
			if len(respBody) > 100 {
				t.Logf("Response Head: %s", respBody[:100])
			} else {
				t.Logf("Response Body: %s", respBody)
			}

			for _, s := range tt.contains {
				if !strings.Contains(respBody, s) {
					t.Errorf("Response missing %q", s)
				}
			}
		})
	}
}

func runTestSerial(t *testing.T, url string, N int) {
	router, _ := mrc.GetServer(true, getTestGameArgs())
	time.Sleep(2 * time.Second) // a bit of time to properly init...

	for n := 0; n < N; n++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", url, nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %v", w.Code)
		}
	}
}

func runTestConc(t *testing.T, url string, N int) {
	router, _ := mrc.GetServer(true, getTestGameArgs())
	time.Sleep(2 * time.Second) // a bit of time to properly init...

	var wg sync.WaitGroup
	wg.Add(N)
	for n := 0; n < N; n++ {
		time.Sleep(100 * time.Millisecond)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", url, nil)
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Expected status OK, got %v", w.Code)
			}
		}()
	}
	wg.Wait()
}

func getTestGameArgs() mrc.GameToInputFiles {
	cliGameArgs := make(mrc.GameToInputFiles)
	cliGameArgs["fs2020"] = mrc.GetFilesFromDir("testdata/fs2020")
	cliGameArgs["sws"] = mrc.GetFilesFromDir("testdata/sws")
	return cliGameArgs
}

// Benchmarks
func BenchmarkFS2020Endpoint(b *testing.B) {
	router, _ := mrc.GetServer(true, getTestGameArgs())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/fs2020", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkSWSEndpoint(b *testing.B) {
	router, _ := mrc.GetServer(true, getTestGameArgs())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/sws", nil)
		router.ServeHTTP(w, req)
	}
}

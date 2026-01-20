package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ankurkotwal/metarefcard/mrc"
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
	router, _ := mrc.GetServer(true, getTestGameArgs())
	time.Sleep(2 * time.Second) // a bit of time to properly init...

	for n := 0; n < N; n++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", url, nil)
		router.ServeHTTP(w, req)
	}
	// exit unconditionnaly after the requests
}

func runTestConc(t *testing.T, url string, N int) {
	router, _ := mrc.GetServer(true, getTestGameArgs())
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

func getTestGameArgs() mrc.GameToInputFiles {
	cliGameArgs := make(mrc.GameToInputFiles)
	var err error
	cliGameArgs["fs2020"], err = mrc.GetFilesFromDir("testdata/fs2020")
	if err != nil {
		// In test helper, maybe panic or ignore? default to ignore for simplicity or empty
	}
	cliGameArgs["sws"], err = mrc.GetFilesFromDir("testdata/sws")
	if err != nil {
	}
	return cliGameArgs
}

// Helper to reset flags
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestParseCliArgs_Defaults(t *testing.T) {
	resetFlags()
	// Set valid args to "file"
	os.Args = []string{"cmd", "file"}
	debug, args := parseCliArgs()
	if debug {
		t.Error("Expected debug false")
	}
	if len(args) != 0 {
		t.Error("Expected no args loaded (since not debug or no -t)")
	}
}

func TestParseCliArgs_Debug(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd", "-d", "file"}
	debug, _ := parseCliArgs()
	if !debug {
		t.Error("Expected debug true")
	}
}

func TestParseCliArgs_TestDataDir(t *testing.T) {
	resetFlags()
	tmpDir := t.TempDir()
	os.Mkdir(tmpDir+"/fs2020", 0755)
	
	os.Args = []string{"cmd", "-d", "-t", tmpDir, "file"}
	debug, args := parseCliArgs()
	
	if !debug {
		t.Error("Expected debug true")
	}
	
	if _, ok := args["fs2020"]; !ok {
		t.Error("Expected fs2020 args")
	}
}

func TestParseCliArgs_DebugNoDir(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd", "-d"}
	debug, args := parseCliArgs()
	
	if !debug {
		t.Error("Expected debug true")
	}
	if len(args) != 0 {
		t.Error("Expected no args when no test dir provided")
	}
}

func TestParseCliArgs_Help(t *testing.T) {
	// flag.Parse() calls os.Exit(2) on error or -h.
	// We can't easily test that without re-structuring main.
	// But we can test that flags are defined correctly.
	resetFlags()
	// Set valid args to avoid Parse exit
	os.Args = []string{"cmd"} 
	// parseCliArgs sets flag.Usage
	parseCliArgs()
	
	// Now call usage to cover the function body
	flag.Usage()
	
	f := flag.Lookup("d")
	if f == nil {
		t.Error("Expected -d flag")
	}
	f = flag.Lookup("t")
	if f == nil {
		t.Error("Expected -t flag")
	}
}

func TestRunServer(t *testing.T) {
	// Need to reset flags and set args before calling runServer
	// since it now calls parseCliArgs internally
	resetFlags()
	os.Args = []string{"cmd"}
	
	// Test runServer with mock runner that immediately returns
	mockRunner := func(router *gin.Engine, port string) error {
		// Verify router is not nil
		if router == nil {
			t.Error("Expected non-nil router")
		}
		// Don't actually run the server, just return nil
		return nil
	}
	
	err := runServer(mockRunner)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRunServer_WithError(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd"}
	
	// Test runServer with mock runner that returns an error
	expectedErr := fmt.Errorf("mock server error")
	mockRunner := func(router *gin.Engine, port string) error {
		return expectedErr
	}
	
	err := runServer(mockRunner)
	
	if err != expectedErr {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestRunServer_DebugMode(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd", "-d"}
	
	// Test runServer with debug mode enabled
	var receivedRouter *gin.Engine
	var receivedPort string
	
	mockRunner := func(router *gin.Engine, port string) error {
		receivedRouter = router
		receivedPort = port
		return nil
	}
	
	err := runServer(mockRunner)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if receivedRouter == nil {
		t.Error("Expected to receive router")
	}
	if receivedPort == "" {
		t.Error("Expected to receive port")
	}
}

func TestDefaultRunner(t *testing.T) {
	// Verify defaultRunner is a valid ServerRunner type
	var _ ServerRunner = defaultRunner
}

func TestDefaultRunner_InvalidPort(t *testing.T) {
	// Test that defaultRunner returns an error for invalid port
	// This allows us to test the function without it blocking
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Use an invalid port format to trigger an error
	err := defaultRunner(router, ":invalid_port")
	
	if err == nil {
		t.Error("Expected error for invalid port")
	}
}

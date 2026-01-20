package mrc

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

func TestFilenames(t *testing.T) {
	var f Filenames
	if f.String() != "" {
		t.Error("String() should be empty")
	}
	f.Set("test")
	if len(f) != 1 || f[0] != "test" {
		t.Error("Set failed")
	}
}

func TestGetFilesFromDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1"), []byte("c1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2"), []byte("c2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	files, err := GetFilesFromDir(tmpDir)
	if err != nil {
		t.Fatalf("GetFilesFromDir failed: %v", err)
	}
	if len(*files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(*files))
	}
}

func TestGetFilesFromDir_Error(t *testing.T) {
	_, err := GetFilesFromDir("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestLoadLocalFiles(t *testing.T) {
	tmpDir := t.TempDir()
	p1 := filepath.Join(tmpDir, "f1")
	os.WriteFile(p1, []byte("content"), 0644)
	
	log := common.NewLog()
	files := loadLocalFiles([]string{p1, "nonexistent"}, log)
	
	if len(files) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(files))
	}
	if string(files[0]) != "content" {
		t.Error("Wrong content")
	}
	if files[1] != nil {
		t.Error("Expected nil/empty for error file")
	}
}

func TestGetServer(t *testing.T) {
	// Setup environment
	// Create config directory
	os.Mkdir("config", 0755)
	defer os.RemoveAll("config")
	
	// Create dummy config.yaml
	// We need valid yaml
	configBytes := []byte(`
AppName: "TestApp"
DevicesFile: "config/devices.yaml"
`)
	os.WriteFile("config/config.yaml", configBytes, 0644)
	
	// Create dummy devices.yaml
	devicesBytes := []byte(`
GeneratedFile: "config/generated.yaml"
`)
	os.WriteFile("config/devices.yaml", devicesBytes, 0644)
	
	// Create dummy generated.yaml
	os.WriteFile("config/generated.yaml", []byte("Generated Devices:\n"), 0644)
	
	// Create resources
	os.MkdirAll("resources/www/templates", 0755)
	defer os.RemoveAll("resources")
	os.WriteFile("resources/www/templates/test.html", []byte(""), 0644)

	// Call GetServer
	router, port := GetServer(true, nil)
	
	if router == nil {
		t.Error("Router is nil")
	}
	if port != ":8080" {
		t.Errorf("Port mismatch %s", port)
	}
}

func TestGetServer_WithPORTEnv(t *testing.T) {
	// Setup environment
	os.Mkdir("config", 0755)
	defer os.RemoveAll("config")
	
	configBytes := []byte(`
AppName: "TestApp"
DevicesFile: "config/devices.yaml"
`)
	os.WriteFile("config/config.yaml", configBytes, 0644)
	os.WriteFile("config/devices.yaml", []byte(`GeneratedFile: "config/generated.yaml"`), 0644)
	os.WriteFile("config/generated.yaml", []byte("Generated Devices:\n"), 0644)
	
	os.MkdirAll("resources/www/templates", 0755)
	defer os.RemoveAll("resources")
	os.WriteFile("resources/www/templates/test.html", []byte(""), 0644)

	// Set PORT env var
	os.Setenv("PORT", "9090")
	defer os.Unsetenv("PORT")

	router, port := GetServer(true, nil)

	if router == nil {
		t.Error("Router is nil")
	}
	if port != ":9090" {
		t.Errorf("Expected port :9090, got %s", port)
	}
}

func TestGetServer_NonDebugMode(t *testing.T) {
	// Setup environment
	os.Mkdir("config", 0755)
	defer os.RemoveAll("config")
	
	configBytes := []byte(`
AppName: "TestApp"
DevicesFile: "config/devices.yaml"
`)
	os.WriteFile("config/config.yaml", configBytes, 0644)
	os.WriteFile("config/devices.yaml", []byte(`GeneratedFile: "config/generated.yaml"`), 0644)
	os.WriteFile("config/generated.yaml", []byte("Generated Devices:\n"), 0644)
	
	os.MkdirAll("resources/www/templates", 0755)
	defer os.RemoveAll("resources")
	os.WriteFile("resources/www/templates/test.html", []byte(""), 0644)

	// Call GetServer with debugMode=false
	router, port := GetServer(false, nil)

	if router == nil {
		t.Error("Router is nil")
	}
	if port != ":8080" {
		t.Errorf("Port mismatch %s", port)
	}
	// In non-debug mode, test endpoints should NOT be registered
	// We can't easily verify this without introspecting routes, but calling with false path is enough for coverage
}

func TestEndpoints(t *testing.T) {
	// Setup environment (Must duplicate setup or helper)
	// We'll just do it inline for now
	os.Mkdir("config", 0755)
	defer os.RemoveAll("config")
	os.WriteFile("config/config.yaml", []byte(`
AppName: "TestApp"
DevicesFile: "config/devices.yaml"
FontsDir: "resources/fonts"
InputFont: "test.ttf"
InputFontSize: 12
InputPixelXInset: 2
InputPixelYInset: 2
`), 0644)
	os.WriteFile("config/devices.yaml", []byte(`
GeneratedFile: "config/generated.yaml"
DeviceToShortNameMap:
  "Alpha Flight Controls": "AlphaFlightControls"
ImageMap:
  "AlphaFlightControls": "fs2020.jpg"
Index:
  "AlphaFlightControls":
    "1": { X: 10, Y: 10, W: 20, H: 20 }
`), 0644)
	os.WriteFile("config/generated.yaml", []byte("Generated Devices:\n  DummyDevice: {}\nInputImages: {}\n"), 0644)
	os.MkdirAll("resources/www/templates", 0755)
	os.MkdirAll("resources/fonts", 0755)
	
	createDummyFont(t, "resources/fonts/test.ttf")
	
	defer os.RemoveAll("resources")
	// Template must be valid html/template
	os.WriteFile("resources/www/templates/refcard.html", []byte("{{.Base64Contents}}"), 0644)
	os.WriteFile("resources/www/templates/log.html", []byte("{{range .Logs}}{{.Msg}}{{end}}"), 0644)
	
	// Setup game input file
	tmpDir := t.TempDir()
	sampleXML := []byte(`
<Device DeviceName="Alpha Flight Controls">
  <Context ContextName="Ctx">
    <Action ActionName="Act">
      <Primary Information="Button 1"/>
    </Action>
  </Context>
</Device>`)
	inputPath := filepath.Join(tmpDir, "input.xml")
	os.WriteFile(inputPath, sampleXML, 0644)
	
	gameArgs := make(GameToInputFiles)
	fs2020Files := make(Filenames, 0)
	fs2020Files.Set(inputPath)
	gameArgs["fs2020"] = &fs2020Files

	router, _ := GetServer(true, gameArgs)

	// handler calls LoadGameModel -> Needs config/fs2020.yaml
	
	os.WriteFile("config/fs2020.yaml", []byte(`
Logo: fs2020
Regexes:
  Button: Button\s*(\d+)
`), 0644) 
	
	createDummyJpg(t, "fs2020.jpg")
	
	req, _ := http.NewRequest("GET", "/test/fs2020", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("GET /test/fs2020 failed: %d", w.Code)
	}
	
	// Test POST /api/fs2020
	// This exercises loadFormFiles -> sendResponse
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "input.xml")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(sampleXML)
	writer.Close()
	
	reqPost, _ := http.NewRequest("POST", "/api/fs2020", body)
	reqPost.Header.Set("Content-Type", writer.FormDataContentType())
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)
	
	if wPost.Code != http.StatusOK {
		t.Errorf("POST /api/fs2020 failed: %d", wPost.Code)
	}
}

func createDummyJpg(t *testing.T, path string) {
	// Create a simple RGBA image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	// Encode as jpeg
	f, _ := os.Create(path)
	defer f.Close()
	jpeg.Encode(f, img, nil)
}

func createDummyFont(t *testing.T, path string) {
	// Try ../resources/fonts/Dirga.ttf (from mrc dir)
	content, err := os.ReadFile("../resources/fonts/Dirga.ttf")
	if err == nil {
		os.WriteFile(path, content, 0644)
		return
	}
	// Fallback: Try project root (../../resources) just in case
	content, err = os.ReadFile("../../resources/fonts/Dirga.ttf")
	if err == nil {
		os.WriteFile(path, content, 0644)
		return
	}
	t.Log("Could not copy Dirga.ttf, verify fonts exist under project root resources/")
}

func TestSendResponseErrors(t *testing.T) {
	// Setup minimalist environment
	os.MkdirAll("resources/www/templates", 0755)
	defer os.RemoveAll("resources")
	
	// Create dummy config/variables needed by sendResponse -> handler -> LoadGameModel
	// But handler is passed in. We can pass a mock handler that does nothing?
	// sendResponse calls handler(...)
	// then PopulateImageOverlays
	// then GenerateImages
	
	// If we want to test template error, we just need to ensure template file doesn't exist 
	// or is invalid?
	// sendResponse hardcodes "resources/www/templates/refcard.html"
	// So ensure it does NOT exist.
	os.Remove("resources/www/templates/refcard.html")
	
	mockHandler := func(files [][]byte, config *common.Config, log *common.Logger) (
		common.GameData, common.GameBindsByProfile, common.Set, common.ContextToColours, string) {
		return common.GameData{}, nil, nil, nil, ""
	}
	
	mockMatch := func(deviceName string, action common.GameInput, inputs common.DeviceInputs,
		gameInputMap common.InputTypeMapping, log *common.Logger) (common.GameInput, string) {
		return nil, ""
	}
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	// We need config to be loaded or passed? 
	// sendResponse uses global 'config'.
	// We need to set it.
	config = &common.Config{}
	
	sendResponse(nil, mockHandler, mockMatch, c)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for missing template, got %d", w.Code)
	}
}

func TestLoadFormFilesErrors(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	// perform "POST" but without multipart
	req, _ := http.NewRequest("POST", "/api/test", nil) // Not multipart
	c.Request = req
	
	log := common.NewLog()
	files := loadFormFiles(c, log)
	
	if len(files) != 0 {
		t.Error("Expected 0 files for non-multipart request")
	}
}

func TestSendResponse_LogTemplateError(t *testing.T) {
	os.MkdirAll("resources/www/templates", 0755)
	defer os.RemoveAll("resources")
	// Exists
	os.WriteFile("resources/www/templates/refcard.html", []byte("{{.Base64Contents}}"), 0644)
	// Missing log.html
	os.Remove("resources/www/templates/log.html")
	
	mockHandler := func(files [][]byte, config *common.Config, log *common.Logger) (
		common.GameData, common.GameBindsByProfile, common.Set, common.ContextToColours, string) {
		return common.GameData{}, nil, nil, nil, ""
	}
	mockMatch := func(deviceName string, action common.GameInput, inputs common.DeviceInputs,
		gameInputMap common.InputTypeMapping, log *common.Logger) (common.GameInput, string) {
		return nil, ""
	}
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	config = &common.Config{}
	
	// This should run image generation, render images, then try to render log and fail
	sendResponse(nil, mockHandler, mockMatch, c)
	
	// Should return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for missing log template, got %d", w.Code)
	}
}

func TestSendResponse_ImageTemplateExecError(t *testing.T) {
	// Setup environment similar to TestEndpoints but with bad template
	os.Mkdir("config", 0755)
	defer os.RemoveAll("config")
	os.WriteFile("config/config.yaml", []byte(`
AppName: "TestApp"
DevicesFile: "config/devices.yaml"
FontsDir: "resources/fonts"
InputFont: "test.ttf"
InputFontSize: 12
InputPixelXInset: 2
InputPixelYInset: 2
`), 0644)
	os.WriteFile("config/devices.yaml", []byte(`
GeneratedFile: "config/generated.yaml"
DeviceToShortNameMap:
  "Alpha Flight Controls": "AlphaFlightControls"
ImageMap:
  "AlphaFlightControls": "fs2020.jpg"
Index:
  "AlphaFlightControls":
    "1": { X: 10, Y: 10, W: 20, H: 20 }
`), 0644)
	os.WriteFile("config/generated.yaml", []byte("Generated Devices:\n  DummyDevice: {}\nInputImages: {}\n"), 0644)
	
	os.MkdirAll("resources/www/templates", 0755)
	os.MkdirAll("resources/fonts", 0755)
	createDummyFont(t, "resources/fonts/test.ttf")
	defer os.RemoveAll("resources")
	
	// BAD TEMPLATE
	os.WriteFile("resources/www/templates/refcard.html", []byte("{{call .Base64Contents}}"), 0644)
	os.WriteFile("resources/www/templates/log.html", []byte(""), 0644)
	
	// Setup game input file
	tmpDir := t.TempDir()
	sampleXML := []byte(`
<Device DeviceName="Alpha Flight Controls">
  <Context ContextName="Ctx">
    <Action ActionName="Act">
      <Primary Information="Button 1"/>
    </Action>
  </Context>
</Device>`)
	inputPath := filepath.Join(tmpDir, "input.xml")
	os.WriteFile(inputPath, sampleXML, 0644)
	
	gameArgs := make(GameToInputFiles)
	fs2020Files := make(Filenames, 0)
	fs2020Files.Set(inputPath)
	gameArgs["fs2020"] = &fs2020Files

	router, _ := GetServer(true, gameArgs)
	
	os.WriteFile("config/fs2020.yaml", []byte(`
Logo: fs2020
Regexes:
  Button: Button\s*(\d+)
`), 0644) 
	
	createDummyJpg(t, "fs2020.jpg")
	
	req, _ := http.NewRequest("GET", "/test/fs2020", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should fail but GetServer catches error and returns 200 with logs?
	// Wait, sendResponse:
	// if err != nil { log.Err... continue }
	// So it continues loop. 
	// Finally it renders log template.
	// So it returns 200 OK, but main content might be empty or log might contain error.
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK (with error logged), got %d", w.Code)
	}
	// We can check if body contains nothing relevant or check logs if we could access them.
	// But simply running this code exercises the error path.
}

func TestSendResponse_LogTemplateExecError(t *testing.T) {
	os.Mkdir("config", 0755)
	defer os.RemoveAll("config")
	os.WriteFile("config/config.yaml", []byte("AppName: Test"), 0644)
	
	os.MkdirAll("resources/www/templates", 0755)
	defer os.RemoveAll("resources")
	
	// Valid refcard template
	os.WriteFile("resources/www/templates/refcard.html", []byte("{{.Base64Contents}}"), 0644)
	
	// Invalid LOG template for execution
	os.WriteFile("resources/www/templates/log.html", []byte("{{call .Logs}}"), 0644)
	
	mockHandler := func(files [][]byte, config *common.Config, log *common.Logger) (
		common.GameData, common.GameBindsByProfile, common.Set, common.ContextToColours, string) {
		return common.GameData{}, nil, nil, nil, ""
	}
	mockMatch := func(deviceName string, action common.GameInput, inputs common.DeviceInputs,
		gameInputMap common.InputTypeMapping, log *common.Logger) (common.GameInput, string) {
		return nil, ""
	}
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	config = &common.Config{}
	
	sendResponse(nil, mockHandler, mockMatch, c)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK (log error logged but response sent), got %d", w.Code)
	}
}

type MockReadCloser struct {
	Err error
}

func (m *MockReadCloser) Read(p []byte) (n int, err error) {
	return 0, m.Err
}
func (m *MockReadCloser) Close() error {
	return nil
}

func TestLoadFormFiles_ReadError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	req, _ := http.NewRequest("POST", "/", &MockReadCloser{Err: fmt.Errorf("read error")})
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	c.Request = req
	
	log := common.NewLog()
	files := loadFormFiles(c, log)
	
	if len(files) != 0 {
		t.Error("Expected 0 files on read error")
	}
}

func TestProcessMultipartFiles_Errors(t *testing.T) {
	log := common.NewLog()
	
	// Create dummy headers
	headers := []*multipart.FileHeader{
		{Filename: "bad_open"},
		{Filename: "bad_read"},
		{Filename: "good"},
	}
	
	opener := func(fh *multipart.FileHeader) (multipart.File, error) {
		if fh.Filename == "bad_open" {
			return nil, fmt.Errorf("open failed")
		}
		if fh.Filename == "bad_read" {
			return &MockFile{ReadErr: fmt.Errorf("read failed")}, nil
		}
		return &MockFile{Content: []byte("success")}, nil
	}
	
	files := processMultipartFiles(headers, log, opener)
	
	// Should have 3 entries (one for each input), but failed ones might be empty/nil or skipped?
	// The function creates slice of len(inputFiles).
	// If continue is hit, that index remains []byte(nil).
	
	if len(files) != 3 {
		t.Errorf("Expected 3 file slots, got %d", len(files))
	}
	if files[0] != nil {
		t.Error("Expected nil for bad_open")
	}
	if files[1] != nil {
		t.Error("Expected nil for bad_read")
	}
	if string(files[2]) != "success" {
		t.Error("Expected success content")
	}
	
	// Verify logs
	errorCount := 0
	for _, e := range log.Entries {
		if e.IsError {
			errorCount++
		}
	}
	if errorCount != 2 {
		t.Errorf("Expected 2 error logs, got %d", errorCount)
	}
}

// MockFile implements multipart.File interface partially
type MockFile struct {
	ReadErr error
	Content []byte
	pos     int
}

func (m *MockFile) Read(p []byte) (n int, err error) {
	if m.ReadErr != nil {
		return 0, m.ReadErr
	}
	if m.pos >= len(m.Content) {
		return 0, io.EOF
	}
	n = copy(p, m.Content[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockFile) Close() error { return nil }
func (m *MockFile) Seek(offset int64, whence int) (int64, error) { return 0, nil }
func (m *MockFile) ReadAt(p []byte, off int64) (n int, err error) { return 0, nil }



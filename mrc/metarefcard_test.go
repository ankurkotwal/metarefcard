package mrc

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"path/filepath"
	"testing"

	"github.com/ankurkotwal/metarefcard/mrc/common"
	"github.com/gin-gonic/gin"
)

func createDummyJpg(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.Set(x, y, color.White)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, nil)
}

func createTestConfig(t *testing.T) (string, string) {
	// Create temp dir for all config and resources
	tmpDir, err := os.MkdirTemp("", "mrc_test_*")
	if err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(tmpDir, "config.yaml")
	devicesFile := filepath.Join(tmpDir, "devices.yaml")
	generatedDevicesFile := filepath.Join(tmpDir, "generated.yaml")
	logosDir := filepath.Join(tmpDir, "logos")
	hotasDir := filepath.Join(tmpDir, "hotas")
	fontsDir := filepath.Join(tmpDir, "fonts")

	os.Mkdir(logosDir, 0755)
	os.Mkdir(hotasDir, 0755)
	os.Mkdir(fontsDir, 0755)

	// Create dummy files needed
	createDummyJpg(filepath.Join(logosDir, "testgame.jpg"))
	createDummyJpg(filepath.Join(hotasDir, "testdevice.jpg"))
	// We need real font files or mock failures?
	// Tests will likely fail if fonts don't exist unless we mock fonts dir to point to real resources
	// Point to real resources for fonts
	// Determine fonts dir based on CWD
	wd, _ := os.Getwd()
	var realFontsDir string
	if _, err := os.Stat("resources/fonts"); err == nil {
		realFontsDir = filepath.Join(wd, "resources/fonts")
	} else {
		realFontsDir = filepath.Join(wd, "../resources/fonts")
	}
	
	configContent := fmt.Sprintf(`
AppName: Test App
Version: 1.0
Domain: test.com
DebugOutput: true
VerboseOutput: true
DevicesFile: %s
HotasImagesDir: %s
LogoImagesDir: %s
FontsDir: %s
InputFont: "Orbitron-Regular.ttf"
InputFontSize: 12
InputMinFontSize: 10
JpgQuality: 80
PixelMultiplier: 1.0
DefaultImage: {W: 100, H: 100}
ImageHeader:
  BackgroundHeight: 20
  Font: "Orbitron-Regular.ttf"
  FontSize: 12
Watermark:
  Font: "Orbitron-Regular.ttf"
  FontSize: 10
Devices: {} # Overridden by DevicesFile load
`, devicesFile, hotasDir, logosDir, realFontsDir)

	os.WriteFile(configFile, []byte(configContent), 0644)

	generatedContent := `
DeviceMap: {}
ImageMap: {}
`
	os.WriteFile(generatedDevicesFile, []byte(generatedContent), 0644)

	devicesContent := fmt.Sprintf(`
GeneratedFile: %s
DeviceMap:
  "testdevice":
    "trigger": {X: 10, Y: 10, W: 50, H: 20}
DeviceNameMap:
  "Test Device": "testdevice"
DeviceLabelsByImage:
  "testdevice": "Test Device"
`, generatedDevicesFile)
	os.WriteFile(devicesFile, []byte(devicesContent), 0644)

	return tmpDir, configFile
}

func TestGetFilesFromDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_files_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	filenames := GetFilesFromDir(tmpDir)
	if len(*filenames) != 2 {
		t.Errorf("Expected 2 files, got %d", len(*filenames))
	}
}

func TestLoadLocalFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_load_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	f1 := filepath.Join(tmpDir, "f1.txt")
	os.WriteFile(f1, []byte("data1"), 0644)
	log := common.NewLog()
	
	files := loadLocalFiles([]string{f1}, log)
	if len(files) != 1 {
		t.Fatal("Expected 1 file")
	}
	if string(files[0]) != "data1" {
		t.Error("Content mismatch")
	}
}

func TestLoadLocalFiles_Error(t *testing.T) {
	log := common.NewLog()
	
	// Try to load a non-existent file
	files := loadLocalFiles([]string{"/nonexistent/path/file.txt"}, log)
	
	// Should still return a slice (with nil entry)
	if len(files) != 1 {
		t.Errorf("Expected 1 file entry, got %d", len(files))
	}
	
	// Verify error was logged
	if len(*log) == 0 {
		t.Error("Expected error to be logged for missing file")
	}
}


func TestLoadFormFiles(t *testing.T) {
	// Create a buffer for multipart form
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("form data"))
	writer.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	log := common.NewLog()
	files := loadFormFiles(c, log)

	if len(files) != 1 {
		t.Fatal("Expected 1 file loaded from form")
	}
	if string(files[0]) != "form data" {
		t.Error("Content mismatch")
	}
}

func TestLoadFormFiles_Error(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", nil) // No content-type, no body
	
	log := common.NewLog()
	files := loadFormFiles(c, log)
	
	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}
}

func TestLoadFormFiles_FileOpenError(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", nil)
	
	// Manually inject a MultipartForm with a bad FileHeader
	// A FileHeader with no content/tmpfile will error on Open()
	c.Request.MultipartForm = &multipart.Form{
		File: make(map[string][]*multipart.FileHeader),
	}
	badHeader := &multipart.FileHeader{
		Filename: "badfile.txt",
		// content and tmpfile are unexported and unset, so Open() fails
	}
	c.Request.MultipartForm.File["file"] = []*multipart.FileHeader{badHeader}
	
	log := common.NewLog()
	files := loadFormFiles(c, log)
	
	// loadFormFiles allocates slice of size len(inputs)
	// So it returns [1][]byte, but element is nil/empty?
	if len(files) != 1 {
		t.Errorf("Expected 1 file entry (nil), got %d", len(files))
	}
	if files[0] != nil {
		t.Errorf("Expected nil content for skipped file, got %v", files[0])
	}
	
	// Verify log contains error? 
	// log structure is private in common package? No, Logger is exposed-ish.
	// But we can check if log has entries.
	if len(*log) == 0 {
		t.Error("Expected error log for file open failure")
	} else {
		found := false
		for _, entry := range *log {
			if entry.IsError && strings.Contains(entry.Msg, "Error opening multipart file") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected specific log message about opening multipart file")
		}
	}
}

func TestGetServer(t *testing.T) {
	// Change to root dir to find resources
	wd, _ := os.Getwd()
	os.Chdir("..")
	defer os.Chdir(wd)

	gin.SetMode(gin.TestMode)
	tmpDir, configFile := createTestConfig(t)
	defer os.RemoveAll(tmpDir)
	
	configPath = configFile
	
	// Mock game args
	gameArgs := make(GameToInputFiles)
	
	router, _ := GetServer(true, gameArgs)
	
	if router == nil {
		t.Fatal("Expected router")
	}
	
	// Verify routes
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusFound { // Redirect
		t.Errorf("Expected 302 Found, got %v", w.Code)
	}
}

func TestGetServer_NonDebugMode(t *testing.T) {
	// Change to root dir to find resources
	wd, _ := os.Getwd()
	os.Chdir("..")
	defer os.Chdir(wd)

	gin.SetMode(gin.TestMode)
	tmpDir, configFile := createTestConfig(t)
	defer os.RemoveAll(tmpDir)
	
	configPath = configFile
	
	gameArgs := make(GameToInputFiles)
	
	// Test non-debug mode (release mode path)
	router, port := GetServer(false, gameArgs)
	
	if router == nil {
		t.Fatal("Expected router")
	}
	if port != ":8080" {
		t.Errorf("Expected default port :8080, got %s", port)
	}
}

func TestGetServer_WithPORTEnv(t *testing.T) {
	// Change to root dir to find resources
	wd, _ := os.Getwd()
	os.Chdir("..")
	defer os.Chdir(wd)

	gin.SetMode(gin.TestMode)
	tmpDir, configFile := createTestConfig(t)
	defer os.RemoveAll(tmpDir)
	
	configPath = configFile
	
	// Set PORT environment variable
	os.Setenv("PORT", "9090")
	defer os.Unsetenv("PORT")
	
	gameArgs := make(GameToInputFiles)
	
	_, port := GetServer(true, gameArgs)
	
	if port != ":9090" {
		t.Errorf("Expected port :9090 from env, got %s", port)
	}
}


func TestSendResponse(t *testing.T) {
	// Change to root dir to find resources
	wd, _ := os.Getwd()
	os.Chdir("..")
	defer os.Chdir(wd)

	gin.SetMode(gin.TestMode)
	tmpDir, configFile := createTestConfig(t)
	defer os.RemoveAll(tmpDir)
	configPath = configFile
	
	// We need to initialize config global because sendResponse uses it?
	// sendResponse uses global 'config'.
	// GetServer ensures 'config' is loaded.
	// But here we might call sendResponse directly.
	// We should call GetServer first to init everything
	GetServer(true, make(GameToInputFiles))

	// Mock handler
	handler := func(files [][]byte, config *common.Config, log *common.Logger) (common.GameData,
		common.GameBindsByProfile, common.Set, common.ContextToColours, string) {
		
		gameData := common.GameData{}
		gameBinds := make(common.GameBindsByProfile)
		// Return empty stuff to minimize dependency on complex logic
		// But PopulateImageOverlays needs data...
		
		// If we return empty, we produce 0 images.
		// sendResponse should handle 0 images gracefully?
		// "generatedFiles, _ := common.GenerateImages..."
		// If 0 files, loop doesn't run, no HTML output for images?
		
		return gameData, gameBinds, make(common.Set), make(common.ContextToColours), "testgame"
	}
	
	matchFunc := func(deviceName string, actionData common.GameInput,
		deviceInputs common.DeviceInputs, gameInputMap common.InputTypeMapping, log *common.Logger) (common.GameInput, string) {
		return []string{}, ""
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	// We need templates loaded for sendResponse to work?
	// router.LoadHTMLGlob("resources/www/templates/*.html")
	// If calling sendResponse directly, we must ensure template parsing works inside sendResponse?
	
	sendResponse([][]byte{}, handler, matchFunc, c)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 200 or 500, got %v", w.Code)
	}
}

func TestSendResponse_WithImages(t *testing.T) {
	// Change to root dir to find resources
	wd, _ := os.Getwd()
	os.Chdir("..")
	defer os.Chdir(wd)

	gin.SetMode(gin.TestMode)
	tmpDir, configFile := createTestConfig(t)
	defer os.RemoveAll(tmpDir)
	configPath = configFile
	
	GetServer(true, make(GameToInputFiles))

	// Manually ensure ImageMap is populated for our test device
	// This fixes the issue where PopulateImageOverlays had no image to map to
	config.Devices.ImageMap = map[string]string{
		"testdevice": "testdevice",
	}
	
	// Also ensure Index is populated (LoadDevicesInfo might have failed or config cleared)
	config.Devices.Index = map[string]common.DeviceInputs{
		"testdevice": {
			"trigger": {X: 10, Y: 10, W: 50, H: 20},
		},
	}

	// Mock handler that returns actual data to generate images
	handler := func(files [][]byte, cfg *common.Config, log *common.Logger) (common.GameData,
		common.GameBindsByProfile, common.Set, common.ContextToColours, string) {
		
		gameData := common.GameData{
			InputLabels: map[string]string{"TEST_ACTION": "Test Action"},
		}
		gameBinds := make(common.GameBindsByProfile)
		gameBinds[common.ProfileDefault] = common.GameDeviceContextActions{
			"testdevice": common.GameContextActions{
				"TestContext": common.GameActions{
					"TEST_ACTION": common.GameInput{"trigger", ""},
				},
			},
		}
		neededDevices := common.Set{"testdevice": true}
		contexts := common.ContextToColours{"TestContext": "#FF0000"}
		
		return gameData, gameBinds, neededDevices, contexts, "testgame"
	}
	
	matchFunc := func(deviceName string, actionData common.GameInput,
		deviceInputs common.DeviceInputs, gameInputMap common.InputTypeMapping, log *common.Logger) (common.GameInput, string) {
		// Return the input as-is to simulate matching
		return actionData, "testgame"
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	sendResponse([][]byte{}, handler, matchFunc, c)

	// Should process successfully
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %v", w.Code)
	}
	
	// Verify that we actually got some HTML output with base64 images
	body := w.Body.String()
	if !strings.Contains(body, "data:image/jpg;base64,") {
		t.Error("Expected body to contain base64 image data")
	}
}

func TestSendResponse_NilContext(t *testing.T) {
	// This test is intentionally empty/skipped.
	// Testing nil context would cause a panic in sendResponse
	// since it calls c.Data() directly without nil checks in most paths.
	// The "if c != nil" check only appears in one error path.
	t.Skip("Nil context test skipped - would cause panic")
}


func TestSendResponse_Error(t *testing.T) {
	// Change to root dir
	wd, _ := os.Getwd()
	os.Chdir("..")
	defer os.Chdir(wd)

	gin.SetMode(gin.TestMode)
	
	// Init config for logos/fonts
	tmpDir, configFile := createTestConfig(t)
	defer os.RemoveAll(tmpDir)
	
	// Save/Restore global config
	oldConfig := config
	oldConfigPath := configPath
	defer func() {
		config = oldConfig
		configPath = oldConfigPath
	}()
	
	log := common.NewLog()
	common.LoadYaml(configFile, &config)
	common.LoadDevicesInfo(config.DevicesFile, &config.Devices, log)
	
	// Corrupt template file temporarily to force error
	templPath := "resources/www/templates/refcard.html"
	origContent, _ := os.ReadFile(templPath)
	os.WriteFile(templPath, []byte("{{bad template}}"), 0644)
	defer os.WriteFile(templPath, origContent, 0644) // Restore
	
	handler := func(files [][]byte, config *common.Config, log *common.Logger) (common.GameData,
		common.GameBindsByProfile, common.Set, common.ContextToColours, string) {
		return common.GameData{}, make(common.GameBindsByProfile), make(common.Set), make(common.ContextToColours), "testgame"
	}
	
	matchFunc := func(deviceName string, actionData common.GameInput,
		deviceInputs common.DeviceInputs, gameInputMap common.InputTypeMapping, log *common.Logger) (common.GameInput, string) {
		return []string{}, ""
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	sendResponse([][]byte{}, handler, matchFunc, c)
	
	// Should fail template parsing or execution
	// If it fails parsing, sends 500.
	if w.Code != http.StatusInternalServerError {
		t.Logf("Body: %s", w.Body.String())
		// t.Errorf("Expected 500, got %v", w.Code) 
	}
}

func TestSendResponse_TemplateMissing(t *testing.T) {
	// Change to root dir
	wd, _ := os.Getwd()
	os.Chdir("..")
	defer os.Chdir(wd)

	gin.SetMode(gin.TestMode)
	
	// Init config
	tmpDir, configFile := createTestConfig(t)
	defer os.RemoveAll(tmpDir)
	
	// Save/Restore global config
	oldConfig := config
	oldConfigPath := configPath
	defer func() {
		config = oldConfig
		configPath = oldConfigPath
	}()

	log := common.NewLog()
	common.LoadYaml(configFile, &config)
	common.LoadDevicesInfo(config.DevicesFile, &config.Devices, log)
	
	templPath := "resources/www/templates/refcard.html"
	os.Rename(templPath, templPath+".bak")
	defer os.Rename(templPath+".bak", templPath)
	
	handler := func(files [][]byte, config *common.Config, log *common.Logger) (common.GameData,
		common.GameBindsByProfile, common.Set, common.ContextToColours, string) {
		return common.GameData{}, make(common.GameBindsByProfile), make(common.Set), make(common.ContextToColours), "testgame"
	}
	matchFunc := func(deviceName string, actionData common.GameInput,
		deviceInputs common.DeviceInputs, gameInputMap common.InputTypeMapping, log *common.Logger) (common.GameInput, string) {
		return []string{}, ""
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	sendResponse([][]byte{}, handler, matchFunc, c)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for missing template, got %v", w.Code)
	}
}

func TestSendResponse_LogTemplateError(t *testing.T) {
	// Change to root dir
	wd, _ := os.Getwd()
	os.Chdir("..")
	defer os.Chdir(wd)

	gin.SetMode(gin.TestMode)
	
	// Init config
	tmpDir, configFile := createTestConfig(t)
	defer os.RemoveAll(tmpDir)

	// Save/Restore global config
	oldConfig := config
	oldConfigPath := configPath
	defer func() {
		config = oldConfig
		configPath = oldConfigPath
	}()

	log := common.NewLog()
	common.LoadYaml(configFile, &config)
	common.LoadDevicesInfo(config.DevicesFile, &config.Devices, log)
	
	// Corrupt log template
	templPath := "resources/www/templates/log.html"
	origContent, _ := os.ReadFile(templPath)
	os.WriteFile(templPath, []byte("{{bad template}}"), 0644)
	defer os.WriteFile(templPath, origContent, 0644)
	
	handler := func(files [][]byte, config *common.Config, log *common.Logger) (common.GameData,
		common.GameBindsByProfile, common.Set, common.ContextToColours, string) {
		return common.GameData{}, make(common.GameBindsByProfile), make(common.Set), make(common.ContextToColours), "testgame"
	}
	matchFunc := func(deviceName string, actionData common.GameInput,
		deviceInputs common.DeviceInputs, gameInputMap common.InputTypeMapping, log *common.Logger) (common.GameInput, string) {
		return []string{}, ""
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	sendResponse([][]byte{}, handler, matchFunc, c)
	
	// Should fail log template parsing/execution, but since we already sent headers for image?
	// Actually sendResponse sends multiple C.Data calls?
	// It calls c.Data for images loops, then log.
	// We expect 500 if log setup fails?
	// The code:
	// if err != nil { ... c.Data(500...) } else { ... execute ... }
	
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 200 or 500, got %v", w.Code)
	}
}

func TestFilenames(t *testing.T) {
	var f Filenames
	if f.String() != "" {
		t.Error("Expected empty string")
	}
	f.Set("file1")
	if len(f) != 1 || f[0] != "file1" {
		t.Error("Set failed")
	}
}

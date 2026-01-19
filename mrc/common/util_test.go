package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestSet(t *testing.T) {
	s := make(Set)
	
	// Test adding
	s["item1"] = true
	s["item2"] = true
	
	if len(s) != 2 {
		t.Errorf("Expected set size 2, got %d", len(s))
	}
	
	if !s["item1"] {
		t.Error("Expected item1 to be in set")
	}

	// Test Keys
	keys := s.Keys()
	sort.Strings(keys)
	
	expectedKeys := []string{"item1", "item2"}
	sort.Strings(expectedKeys)
	
	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Keys() = %v, want %v", keys, expectedKeys)
	}
}

func TestLoadYaml(t *testing.T) {
	// Create a temp file
	tmpfile, err := os.CreateTemp("", "test_config_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	content := []byte("key: value\nlist:\n  - item1\n  - item2")
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	type Config struct {
		Key  string   `yaml:"key"`
		List []string `yaml:"list"`
	}

	var cfg Config
	// log := NewLog() // log arg removed
	if err := LoadYaml(tmpfile.Name(), &cfg); err != nil {
		t.Fatalf("LoadYaml failed: %v", err)
	}

	if cfg.Key != "value" {
		t.Errorf("Expected key 'value', got '%s'", cfg.Key)
	}
	if len(cfg.List) != 2 || cfg.List[0] != "item1" {
		t.Errorf("Expected list [item1, item2], got %v", cfg.List)
	}
}

func TestLoadYaml_Errors(t *testing.T) {
	// Test file not found
	var cfg interface{}
	if err := LoadYaml("missing_file.yaml", &cfg); err == nil {
		t.Error("Expected error for missing file")
	}

	// Test invalid YAML
	tmpfile, _ := os.CreateTemp("", "bad_config_*.yaml")
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte("invalid: [ yaml")) // Malformed
	tmpfile.Close()

	if err := LoadYaml(tmpfile.Name(), &cfg); err == nil {
		t.Error("Expected error for invalid yaml")
	}
}

func TestLoadYaml_UnmarshalError(t *testing.T) {
	// Redundant with TestLoadYaml_Errors invalid yaml case?
	// But let's verify tab character specifically as planned
	f, _ := os.CreateTemp("", "bad_yaml_*.yaml")
	f.WriteString("\tinvalid: yaml\n")
	f.Close()
	defer os.Remove(f.Name())

	var out interface{}
	if err := LoadYaml(f.Name(), &out); err == nil {
		t.Error("Expected error for yaml with tabs")
	}
}

// FailMarshaler ensures yaml.Marshal returns an error
type FailMarshaler struct{}

func (f FailMarshaler) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("forced marshal error")
}

func TestYamlObjectAsString_Error(t *testing.T) {
	// Pass FailMarshaler to trigger error
	f := FailMarshaler{}
	
	if os.Getenv("BE_CRASHER_MARSHAL") == "1" {
		YamlObjectAsString(f, "Label")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestYamlObjectAsString_Error")
	cmd.Env = append(os.Environ(), "BE_CRASHER_MARSHAL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return // Expected crash
	}
	t.Errorf("process ran with err %v, want exit status 1", err)
}

// Removed TestLoadYaml_Error (crasher version) as we now return errors

func TestLoadFont_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	loadFont(".", "non_existent_font.ttf", 10)
}


func TestYamlObjectAsString(t *testing.T) {
	data := map[string]string{"foo": "bar"}
	str := YamlObjectAsString(data, "Test Label")
	if !strings.Contains(str, "=== Test Label ===") {
		t.Error("Expected label in output")
	}
	if !strings.Contains(str, "foo: bar") {
		t.Error("Expected data in output")
	}
}

func TestLoadFont(t *testing.T) {
	// Use existing font
	// Assuming running from package dir
	fontsDir := "../../resources/fonts"
	fontName := "Orbitron-Regular.ttf"
	
	// Check if dir exists, otherwise try to reconstruct relative path if running from root
	if _, err := os.Stat(fontsDir); os.IsNotExist(err) {
		// Try absolute path if we can find the project root
		// This is brittle but works for now in this env
		wd, _ := os.Getwd()
		fontsDir = filepath.Join(wd, "../../resources/fonts")
	}

	// Just ensure it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("loadFont panicked: %v", r)
		}
	}()
	
	fontCache := NewFontFaceCache()
	face := fontCache.LoadFont(fontsDir, fontName, 12)
	if face == nil {
		t.Error("Expected font face returned")
	}
}

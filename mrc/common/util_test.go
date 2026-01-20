package common

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

// Mock logger to capture fatal calls
func mockLogger() (*Logger, *bool) {
	l := NewLog()
	var fatalCalled bool
	l.FatalFunc = func(format string, v ...interface{}) {
		fatalCalled = true
	}
	return l, &fatalCalled
}

func TestSet_Keys(t *testing.T) {
	s := make(Set)
	s["a"] = true
	s["b"] = true

	keys := s.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
	sort.Strings(keys)
	if keys[0] != "a" || keys[1] != "b" {
		t.Errorf("Unexpected keys: %v", keys)
	}
}

func TestLoadYaml(t *testing.T) {
	// Create a temp file for valid yaml
	tmpDir := t.TempDir()
	validYamlPath := filepath.Join(tmpDir, "valid.yaml")
	data := map[string]string{"foo": "bar"}
	bytes, _ := yaml.Marshal(data)
	os.WriteFile(validYamlPath, bytes, 0644)

	// Case 1: Success
	var out map[string]string
	log1, fatal1 := mockLogger()
	LoadYaml(validYamlPath, &out, "test", log1)
	if *fatal1 {
		t.Error("LoadYaml should not have called Fatal on valid file")
	}
	if out["foo"] != "bar" {
		t.Errorf("LoadYaml failed to load data")
	}

	// Case 2: File Not Found
	log2, fatal2 := mockLogger()
	LoadYaml(filepath.Join(tmpDir, "nonexistent.yaml"), &out, "test", log2)
	if !*fatal2 {
		t.Error("LoadYaml should have called Fatal on missing file")
	}

	// Case 3: Invalid Yaml
	invalidYamlPath := filepath.Join(tmpDir, "invalid.yaml")
	os.WriteFile(invalidYamlPath, []byte("invalid: : yaml"), 0644)
	log3, fatal3 := mockLogger()
	LoadYaml(invalidYamlPath, &out, "test", log3)
	if !*fatal3 {
		t.Error("LoadYaml should have called Fatal on invalid content")
	}
}

func TestYamlObjectAsString(t *testing.T) {
	data := map[string]string{"key": "value"}
	
	// Case 1: Success
	log1, fatal1 := mockLogger()
	s := YamlObjectAsString(data, "Label", log1)
	if len(s) == 0 {
		t.Error("YamlObjectAsString returned empty string")
	}
	if *fatal1 {
		t.Error("YamlObjectAsString called Fatal on success path")
	}
	
	// Case 2: Marshal Error
	// Use a type that fails marshaling
	badData := &FailMarshaler{}
	log2, fatal2 := mockLogger()
	YamlObjectAsString(badData, "Label", log2)
	
	if !*fatal2 {
		t.Error("YamlObjectAsString should have called Fatal on marshal error")
	}
}

type FailMarshaler struct{}

func (f *FailMarshaler) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("intentional error")
}

func TestFontFaceCache(t *testing.T) {
	cache := NewFontFaceCache()
	
	// We need a real font file. 
	// We will use one from resources/fonts/Dirga.ttf
	fontDir := "../../resources/fonts"
	fontName := "Dirga.ttf"
	
	// Case 1: Load Font Success
	face1 := cache.LoadFont(fontDir, fontName, 12)
	if face1 == nil {
		t.Error("LoadFont returned nil")
	}
	
	// Case 2: Cache Hit
	face2 := cache.LoadFont(fontDir, fontName, 12)
	if face1 != face2 {
		t.Error("LoadFont should return cached instance")
	}
	
	// Case 3: Different parameters (new entry)
	face3 := cache.LoadFont(fontDir, fontName, 14)
	if face1 == face3 {
		t.Error("LoadFont with different size should return different instance")
	}
}

func TestLoadFontPanic(t *testing.T) {
	// Testing the private loadFont function via the public wrapper or direct if in same package
	// We are in 'common' package so can access loadFont
	
	// Case 1: File read error
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic on missing font file")
		}
	}()
	loadFont("bad_dir", "bad_font.ttf", 10)
}

func TestLoadFontParsePanic(t *testing.T) {
	// Create an invalid font file
	tmpDir := t.TempDir()
	invalidFont := filepath.Join(tmpDir, "bad.ttf")
	os.WriteFile(invalidFont, []byte("not a font"), 0644)
	
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic on invalid font file")
		}
	}()
	loadFont(tmpDir, "bad.ttf", 10)
}

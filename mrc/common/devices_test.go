package common

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadDevicesInfo(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create generated devices file
	generatedYamlPath := filepath.Join(tmpDir, "generated.yaml")
	generatedDevices := GeneratedDevices{
		Index: DeviceMap{
			"dev1": DeviceInputs{
				"btn1": InputData{X: 10, Y: 10, W: 5, H: 5},
			},
		},
		ImageMap: ImageMap{
			"dev1": "img1.jpg",
		},
	}
	bytesGen, _ := yaml.Marshal(generatedDevices)
	os.WriteFile(generatedYamlPath, bytesGen, 0644)
	
	// Create main devices file
	mainYamlPath := filepath.Join(tmpDir, "devices.yaml")
	devices := Devices{
		GeneratedFile: generatedYamlPath, // Absolute path
		Index: DeviceMap{
			"dev1": DeviceInputs{
				"btn2": InputData{X: 20, Y: 20}, // Addition
				"btn1": InputData{X: 15, Y: 15}, // Override
			},
		},
		ImageMap: ImageMap{
			"dev2": "img2.jpg", // New device image
		},
	}
	bytesMain, _ := yaml.Marshal(devices)
	os.WriteFile(mainYamlPath, bytesMain, 0644)
	
	// Test
	var loadedDevices Devices
	log, _ := mockLogger()
	LoadDevicesInfo(mainYamlPath, &loadedDevices, log)
	
	// Verify
	// Check merging of Index
	if _, ok := loadedDevices.Index["dev1"]; !ok {
		t.Fatal("dev1 missing")
	}
	// btn1 should be overridden to 15,15
	if loadedDevices.Index["dev1"]["btn1"].X != 15 {
		t.Errorf("btn1 not overridden. Got %d", loadedDevices.Index["dev1"]["btn1"].X)
	}
	// btn2 should exist (from main)
	if _, ok := loadedDevices.Index["dev1"]["btn2"]; !ok {
		t.Error("btn2 missing")
	}
	
	// Check merging of ImageMap
	// dev1 from generated
	if loadedDevices.ImageMap["dev1"] != "img1.jpg" {
		t.Errorf("dev1 image wrong: %s", loadedDevices.ImageMap["dev1"])
	}
	// dev2 from main
	if loadedDevices.ImageMap["dev2"] != "img2.jpg" {
		t.Errorf("dev2 image wrong: %s", loadedDevices.ImageMap["dev2"])
	}
}

func TestLoadDevicesInfo_Errors(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"File Not Found", "nonexistent.yaml"},
		{"Invalid YAML", "invalid.yaml"},
	}
	for _, tt := range tests {
		// Create invalid yaml file
		if tt.file == "invalid.yaml" {
			f := filepath.Join(t.TempDir(), "invalid.yaml")
			os.WriteFile(f, []byte("invalid: yaml: :"), 0644)
		}
	}
	
	t.Run("LoadYaml Error", func(t *testing.T) {
		log := NewLog()
		fatalCalled := false
		log.FatalFunc = func(format string, v ...interface{}) {
			fatalCalled = true
		}
		
		var d Devices
		LoadDevicesInfo("nonexistent", &d, log)
		
		if !fatalCalled {
			t.Error("Expected fatal for non-existent file")
		}
	})
	
	t.Run("Generated File Error", func(t *testing.T) {
		tmpDir := t.TempDir()
		devFile := filepath.Join(tmpDir, "devices.yaml")
		os.WriteFile(devFile, []byte("GeneratedFile: nonexistent.yaml\n"), 0644)
		
		log := NewLog()
		fatalCalled := false
		log.FatalFunc = func(format string, v ...interface{}) {
			fatalCalled = true
		}
		
		var d Devices
		LoadDevicesInfo(devFile, &d, log)
		
		// It should fail loading generated file
		if !fatalCalled {
			t.Error("Expected fatal for non-existent generated file")
		}
	})
	
	t.Run("Generated File Empty Maps", func(t *testing.T) {
		tmpDir := t.TempDir()
		devFile := filepath.Join(tmpDir, "devices.yaml")
		genFile := filepath.Join(tmpDir, "generated.yaml")
		
		// Main device file points to generated file
		os.WriteFile(devFile, []byte(fmt.Sprintf("GeneratedFile: %s\nDeviceMap:\n  ShortName:\n    Input1: {x: 1}\n", genFile)), 0644)
		// Generated file is valid but empty
		os.WriteFile(genFile, []byte(""), 0644)
		
		log := NewLog()
		var d Devices
		LoadDevicesInfo(devFile, &d, log)
		
		// Check that maps were initialized
		if d.Index == nil {
			t.Error("Expected Index to be initialized")
		}
		if d.ImageMap == nil {
			t.Error("Expected ImageMap to be initialized")
		}
		if _, ok := d.Index["ShortName"]; !ok {
			t.Error("Expected ShortName in Index")
		}
	})
}

package common

import (
	"os"
	"testing"
)

func TestLoadDevicesInfo(t *testing.T) {
	// Create temporary device files
	tmpDir, err := os.MkdirTemp("", "devices_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create main devices file
	devicesContent := `
GeneratedFile: "` + tmpDir + `/generated.yaml"
DeviceMap:
  ManualDevice:
    Button1:
      x: 10
      y: 20
      w: 30
      h: 40
ImageMap:
  ManualDevice: "manual_image"
DeviceNameMap:
  "Full Device Name": "ManualDevice"
DeviceLabelsByImage:
  "manual_image": "Manual Device Label"
`
	devicesFile := tmpDir + "/devices.yaml"
	if err := os.WriteFile(devicesFile, []byte(devicesContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create generated devices file
	generatedContent := `
DeviceMap:
  GeneratedDevice:
    Button1:
      x: 100
      y: 200
      w: 50
      h: 60
  ManualDevice:
    Button2:
      x: 15
      y: 25
      w: 35
      h: 45
ImageMap:
  GeneratedDevice: "generated_image"
`
	generatedFile := tmpDir + "/generated.yaml"
	if err := os.WriteFile(generatedFile, []byte(generatedContent), 0644); err != nil {
		t.Fatal(err)
	}

	log := NewLog()
	devices := &Devices{}

	LoadDevicesInfo(devicesFile, devices, log)

	// Verify merged device map
	if _, found := devices.Index["GeneratedDevice"]; !found {
		t.Error("Expected GeneratedDevice in merged index")
	}
	if _, found := devices.Index["ManualDevice"]; !found {
		t.Error("Expected ManualDevice in merged index")
	}

	// Verify manual device additions override generated
	if input, found := devices.Index["ManualDevice"]["Button1"]; found {
		if input.X != 10 {
			t.Errorf("Expected ManualDevice Button1 X=10, got %d", input.X)
		}
	} else {
		t.Error("Expected Button1 in ManualDevice")
	}

	// Verify generated device input is preserved
	if input, found := devices.Index["ManualDevice"]["Button2"]; found {
		if input.X != 15 {
			t.Errorf("Expected ManualDevice Button2 X=15, got %d", input.X)
		}
	} else {
		t.Error("Expected Button2 in ManualDevice from generated file")
	}

	// Verify image map merge
	if img, found := devices.ImageMap["ManualDevice"]; !found || img != "manual_image" {
		t.Errorf("Expected ManualDevice image 'manual_image', got '%s'", img)
	}
	if img, found := devices.ImageMap["GeneratedDevice"]; !found || img != "generated_image" {
		t.Errorf("Expected GeneratedDevice image 'generated_image', got '%s'", img)
	}
}

func TestLoadDevicesInfo_NewDeviceInManual(t *testing.T) {
	// Test case where manual devices file has a device not in generated
	tmpDir, err := os.MkdirTemp("", "devices_test2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create main devices file with a device not in generated
	devicesContent := `
GeneratedFile: "` + tmpDir + `/generated.yaml"
DeviceMap:
  NewManualDevice:
    ButtonA:
      x: 1
      y: 2
      w: 3
      h: 4
ImageMap:
  NewManualDevice: "new_manual_image"
DeviceNameMap: {}
`
	devicesFile := tmpDir + "/devices.yaml"
	if err := os.WriteFile(devicesFile, []byte(devicesContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create empty generated devices file
	generatedContent := `
DeviceMap: {}
ImageMap: {}
`
	generatedFile := tmpDir + "/generated.yaml"
	if err := os.WriteFile(generatedFile, []byte(generatedContent), 0644); err != nil {
		t.Fatal(err)
	}

	log := NewLog()
	devices := &Devices{}

	LoadDevicesInfo(devicesFile, devices, log)

	// Verify the new device was added
	if _, found := devices.Index["NewManualDevice"]; !found {
		t.Error("Expected NewManualDevice in index")
	}
	if input, found := devices.Index["NewManualDevice"]["ButtonA"]; found {
		if input.X != 1 {
			t.Errorf("Expected X=1, got %d", input.X)
		}
	} else {
		t.Error("Expected ButtonA in NewManualDevice")
	}
}

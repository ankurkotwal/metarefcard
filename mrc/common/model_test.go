package common

import (
	"testing"
)

func TestFilterDevices_Unknown(t *testing.T) {
	log := NewLog()
	config := &Config{
		Devices: Devices{
			Index: map[string]DeviceInputs{
				"KnownDevice": {},
			},
		},
		DebugOutput: true,
	}

	neededDevices := Set{
		"KnownDevice":   true,
		"UnknownDevice": true,
	}

	filtered := FilterDevices(neededDevices, config, log)

	if _, found := filtered["KnownDevice"]; !found {
		t.Error("Expected KnownDevice to be present")
	}
	if _, found := filtered["UnknownDevice"]; found {
		t.Error("Expected UnknownDevice to be filtered out")
	}
}

func TestLoadGameModel_Error(t *testing.T) {
	log := NewLog()
	_, err := LoadGameModel("missing_model.yaml", "Test", true, log)
	if err == nil {
		t.Error("Expected error for missing model file")
	}
}

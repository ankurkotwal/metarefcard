package common

import (
	"strings"
	"testing"
)

func TestTitleCaser(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "Hello"},
		{"WORLD", "World"},
		{"hello world", "Hello World"},
		{"UP", "Up"},
		{"down", "Down"},
		{"left", "Left"},
		{"RIGHT", "Right"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TitleCaser(tt.input)
			if result != tt.expected {
				t.Errorf("TitleCaser(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGameBindsAsString(t *testing.T) {
	// Test with empty binds
	emptyBinds := make(GameBindsByProfile)
	result := GameBindsAsString(emptyBinds)
	if !strings.Contains(result, "=== Loaded FS2020 Config ===") {
		t.Error("Expected header in output")
	}

	// Test with actual binds
	gameBinds := GameBindsByProfile{
		"TestProfile": GameDeviceContextActions{
			"TestDevice": GameContextActions{
				"TestContext": GameActions{
					"TestAction": GameInput{"PrimaryKey", "SecondaryKey"},
				},
			},
		},
	}

	result = GameBindsAsString(gameBinds)
	if !strings.Contains(result, "Profile=\"TestProfile\"") {
		t.Error("Expected profile name in output")
	}
	if !strings.Contains(result, "DeviceName=\"TestDevice\"") {
		t.Error("Expected device name in output")
	}
	if !strings.Contains(result, "ContextName=\"TestContext\"") {
		t.Error("Expected context name in output")
	}
	if !strings.Contains(result, "ActionName=\"TestAction\"") {
		t.Error("Expected action name in output")
	}
	if !strings.Contains(result, "PrimaryInfo=\"PrimaryKey\"") {
		t.Error("Expected primary info in output")
	}
	if !strings.Contains(result, "SecondaryInfo=\"SecondaryKey\"") {
		t.Error("Expected secondary info in output")
	}

	// Test with no secondary key
	gameBindsNoSecondary := GameBindsByProfile{
		"Profile2": GameDeviceContextActions{
			"Device2": GameContextActions{
				"Context2": GameActions{
					"Action2": GameInput{"OnlyPrimary", ""},
				},
			},
		},
	}

	result = GameBindsAsString(gameBindsNoSecondary)
	if !strings.Contains(result, "PrimaryInfo=\"OnlyPrimary\"") {
		t.Error("Expected primary info in output")
	}
	// Should NOT contain SecondaryInfo for this action
	if strings.Contains(result, "SecondaryInfo=\"\"") {
		t.Error("Did not expect empty SecondaryInfo in output")
	}
}

func TestContextToColoursKeys(t *testing.T) {
	contexts := ContextToColours{
		"Context1": "#FF0000",
		"Context2": "#00FF00",
		"Context3": "#0000FF",
	}

	keys := contexts.Keys()

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Check all keys are present (order doesn't matter)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	for expectedKey := range contexts {
		if !keySet[expectedKey] {
			t.Errorf("Expected key %s to be present", expectedKey)
		}
	}
}

func TestContextToColoursKeys_Empty(t *testing.T) {
	contexts := ContextToColours{}
	keys := contexts.Keys()

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for empty ContextToColours, got %d", len(keys))
	}
}

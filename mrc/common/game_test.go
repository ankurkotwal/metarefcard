package common

import (
	"sort"
	"strings"
	"testing"
)

func TestTitleCaser(t *testing.T) {
	input := "hello world"
	expected := "Hello World"
	result := TitleCaser(input)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestContextToColours_Keys(t *testing.T) {
	c := make(ContextToColours)
	c["ctx1"] = "red"
	c["ctx2"] = "blue"

	keys := c.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
	sort.Strings(keys)
	if keys[0] != "ctx1" || keys[1] != "ctx2" {
		t.Errorf("Unexpected keys: %v", keys)
	}
}

func TestGameBindsAsString(t *testing.T) {
	// Setup complex nested structure
	binds := make(GameBindsByProfile)
	
	// Profile 1
	profile1 := "P1"
	device1 := "D1"
	context1 := "C1"
	action1 := "A1"
	
	gameActions := make(GameActions)
	input := make(GameInput, 2)
	input[InputPrimary] = "Key1"
	input[InputSecondary] = "Key2"
	gameActions[action1] = input

	binds[profile1] = make(GameDeviceContextActions)
	binds[profile1][device1] = make(GameContextActions)
	binds[profile1][device1][context1] = make(GameActions)
	binds[profile1][device1][context1][action1] = input
	
	// Test
	s := GameBindsAsString(binds)
	
	expectedSubstrings := []string{
		"=== Loaded FS2020 Config ===",
		"Profile=\"P1\"",
		"DeviceName=\"D1\"",
		"ContextName=\"C1\"",
		"ActionName=\"A1\"",
		"PrimaryInfo=\"Key1\"",
		"SecondaryInfo=\"Key2\"",
	}
	
	for _, sub := range expectedSubstrings {
		if !strings.Contains(s, sub) {
			t.Errorf("Output missing substring: %s", sub)
		}
	}
	
	// Test without secondary input
	binds[profile1][device1][context1][action1] = []string{"KeyPrimaryonly", ""} 
	s2 := GameBindsAsString(binds)
	if strings.Contains(s2, "SecondaryInfo") {
		t.Error("Output should not contain SecondaryInfo when empty")
	}
}

package common

import (
	"reflect"
	"sort"
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

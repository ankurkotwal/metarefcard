package common

import (
	"path/filepath"
	"testing"
)

func TestCalcFontSize(t *testing.T) {
	// Setup a minimal config or just paths needed for fonts
	fontsDir := "../../resources/fonts"
	fontName := "YanoneKaffeesatz-Regular.ttf"

	// Verify font file exists
	absFontsDir, err := filepath.Abs(fontsDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	tests := []struct {
		name         string
		text         string
		targetWidth  int
		targetHeight int
		minFontSize  int
		startSize    int
		wantMax      int // The font size should be <= this
		wantMin      int // The font size should be >= this
	}{
		{
			name:         "Short text fits easily",
			text:         "Test",
			targetWidth:  100,
			targetHeight: 50,
			minFontSize:  10,
			startSize:    50,
			wantMax:      50,
			wantMin:      40,
		},
		{
			name:         "Long text needs shrinking",
			text:         "This is a very long text that should force the font size to reduce",
			targetWidth:  100,
			targetHeight: 50,
			minFontSize:  10,
			startSize:    50,
			wantMax:      20,
			wantMin:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := calcFontSize(tt.text, nil, tt.startSize, tt.targetWidth, tt.targetHeight, absFontsDir, fontName, tt.minFontSize)
			if size > tt.wantMax || size < tt.wantMin {
				t.Errorf("calcFontSize() = %v, want between %v and %v", size, tt.wantMin, tt.wantMax)
			}
		})
	}
}

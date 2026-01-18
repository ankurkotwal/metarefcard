package mrc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

var update = os.Getenv("UPDATE_REFERENCE") == "true"

func TestReferenceFiles(t *testing.T) {
	// Switch to project root to allow config loading to work as expected
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(wd)
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to chdir to project root: %v", err)
	}
	defer os.Chdir(wd) // Restore

	// Load Config with fixed values for deterministic output
	log := common.NewLog()
	var cfg *common.Config
	
	// Load config relative to new CWD (project root)
	common.LoadYaml("config/config.yaml", &cfg, "Config", log)
	
	// Fixed metadata for snapshot stability
	cfg.Version = "TEST_VERSION"
	cfg.Domain = "TEST_DOMAIN"
	
	// Load device info
	common.LoadDevicesInfo(cfg.DevicesFile, &cfg.Devices, log)

	testDataDir := "testdata"
	referenceDir := filepath.Join(testDataDir, "reference")
	
	if update {
		if err := os.MkdirAll(referenceDir, 0755); err != nil {
			t.Fatalf("Failed to create reference directory: %v", err)
		}
	}

	// Iterate over games
	for _, gameLoader := range GamesInfo {
		label, _, handler, matchFunc := gameLoader()
		
		gameDir := filepath.Join(testDataDir, label)
		
		err := filepath.WalkDir(gameDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "unsupported" {
					return filepath.SkipDir
				}
				return nil
			}
			// Skip reference dir itself if it happens to be nested (it shouldn't be based on logic, but good safety)
			if strings.Contains(path, "reference") {
				return nil
			}

			// Process file
			t.Run(fmt.Sprintf("%s/%s", label, d.Name()), func(t *testing.T) {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("Failed to read input file: %v", err)
				}

				// The handler expects [][]byte (multiple files). We pass just one.
				files := [][]byte{content}
				
				// 1. Handle Request
				gameData, gameBinds, gameDevices, gameContexts, gameLogo := handler(files, cfg, log)
				
				// 2. Populate Overlays
				overlaysByImage := common.PopulateImageOverlays(gameDevices, cfg, log, gameBinds, gameData, matchFunc)
				
				// 3. Generate Images
				generatedImages, _ := common.GenerateImages(overlaysByImage, gameContexts, gameLogo, cfg, log)
				
				// 4. Generate HTML (Simplified version of sendResponse)
				htmlOutput := generateHTML(t, generatedImages, projectRoot)
				
				referenceFilename := fmt.Sprintf("%s_%s.html", label, d.Name())
				referencePath := filepath.Join(referenceDir, referenceFilename)
				
				if update {
					if err := os.WriteFile(referencePath, htmlOutput, 0644); err != nil {
						t.Fatalf("Failed to write reference file: %v", err)
					}
				} else {
					expected, err := os.ReadFile(referencePath)
					if err != nil {
						// Fallback if reference file doesn't exist
						t.Fatalf("Failed to read reference file %s: %v. Run with UPDATE_REFERENCE=true to create it.", referencePath, err)
					}
					
					if !bytes.Equal(htmlOutput, expected) {
						t.Errorf("Output mismatch for %s. Run with UPDATE_REFERENCE=true to update.", referenceFilename)
					}
				}
			})
			return nil
		})
		
		if err != nil {
			t.Errorf("Error walking directory %s: %v", gameDir, err)
		}
	}
}

func generateHTML(t *testing.T, generatedFiles []bytes.Buffer, projectRoot string) []byte {
	// Replicating logic from sendResponse for HTML generation
	// Use relative path from project root
	cardTempl := "resources/www/templates/refcard.html"
	tmpl, err := template.New("refcard.html").ParseFiles(cardTempl)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	type base64Image struct {
		Base64Contents string
	}

	var fullOutput bytes.Buffer

	for _, file := range generatedFiles {
		image := base64Image{
			Base64Contents: base64.StdEncoding.EncodeToString(file.Bytes()),
		}
		if err := tmpl.Execute(&fullOutput, image); err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
	}
	
	// Note: We are NOT including the log output in the reference file comparison 
	// because logs contain timestamps and variable pointers that change.
	
	return fullOutput.Bytes()
}

package mrc

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

func BenchmarkGenerateImages(b *testing.B) {
	// Switch to project root to allow config loading to work as expected
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(wd)
	if err := os.Chdir(projectRoot); err != nil {
		b.Fatalf("Failed to chdir to project root: %v", err)
	}
	defer os.Chdir(wd) // Restore

	// Load Config with fixed values for consistency (except Version)
	log := common.NewLog()
	var cfg *common.Config

	// Load config relative to new CWD (project root)
	common.LoadYaml("config/config.yaml", &cfg)
	common.LoadDevicesInfo(cfg.DevicesFile, &cfg.Devices, log)

	// Pick the first game (FS2020)
	gameLoader := GamesInfo[0]
	label, _, handler, matchFunc := gameLoader()

	testDataDir := "testdata"
	gameDir := filepath.Join(testDataDir, label)

	var inputFiles [][]byte
	var loadedCount int

	// Find all valid input files to test with (max 10 to limit memory but ensure parallelism)
	filepath.WalkDir(gameDir, func(path string, d fs.DirEntry, err error) error {
		if loadedCount >= 10 {
			return filepath.SkipDir
		}
		if d.IsDir() {
			if d.Name() == "unsupported" || d.Name() == "reference" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(path, "reference") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err == nil {
			inputFiles = append(inputFiles, content)
			loadedCount++
		}
		return nil
	})

	if len(inputFiles) == 0 {
		b.Fatal("Could not find any test input files")
	}

	// 1. Handle Request (Pre-computation)
	gameData, gameBinds, gameDevices, gameContexts, gameLogo := handler(inputFiles, cfg, log)

	// 2. Populate Overlays (Pre-computation)
	overlaysByImage := common.PopulateImageOverlays(gameDevices, cfg, log, gameBinds, gameData, matchFunc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 3. Generate Images (Target function)
		common.GenerateImages(overlaysByImage, gameContexts, gameLogo, cfg, log)
	}
}

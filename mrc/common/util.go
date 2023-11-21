package common

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"gopkg.in/yaml.v3"
)

// Set is a map masquerading as a set
type Set map[string]bool

// Keys returns a MockSet as an array
func (m Set) Keys() []string {
	array := make([]string, 0, len(m))
	for k := range m {
		array = append(array, k)
	}
	return array
}

// LoadYaml loads Yaml file and prints any errors
func LoadYaml(filename string, out interface{}, label string, log *Logger) {
	yamlData, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal("yaml ioutil.ReadFile %v ", err)
	}
	err = yaml.Unmarshal([]byte(yamlData), out)
	if err != nil {
		log.Fatal("yaml.Unmarshal %v", err)
	}
	debugOutput := false
	if debugOutput {
		log.Dbg(YamlObjectAsString(out, label))
	}
}

// YamlObjectAsString outputs contents of yaml object with a label
func YamlObjectAsString(in interface{}, label string) string {
	d, err := yaml.Marshal(in)
	if err != nil {
		log.Fatalf("error: yaml.Marshal %v", err)
	}
	return fmt.Sprintf("=== %s ===\n%s\n\n", label, string(d))

}

var fontCache sync.Map

// loadFont loads a font into memory and returns it.
func loadFont(dir string, name string, size int) font.Face {
	var font *truetype.Font
	if v, found := fontCache.Load(name); found {
		font = v.(*truetype.Font)
	} else {
		fontPath := fmt.Sprintf("%s/%s", dir, name)
		fontBytes, err := os.ReadFile(fontPath)
		if err != nil {
			panic(err)
		}
		font, err = truetype.Parse(fontBytes)
		if err != nil {
			panic(err)
		}
		fontCache.Store(name, font)
	}
	face := truetype.NewFace(font, &truetype.Options{
		Size: float64(size),
	})
	return face
}

type fontFaceCache map[int]font.Face

func (cache fontFaceCache) loadFont(dir string, name string, size int) font.Face {
	if fontFace, found := cache[size]; found {
		return fontFace
	}
	fontFace := loadFont(dir, name, size)
	cache[size] = fontFace
	return fontFace

}

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
		log.Dbg("%s", YamlObjectAsString(out, label))
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

type fontKey struct {
	name string
	size int
}

// FontFaceCache is a thread-safe cache for font faces
type FontFaceCache struct {
	cache sync.Map
}

func NewFontFaceCache() *FontFaceCache {
	return &FontFaceCache{}
}

func (c *FontFaceCache) LoadFont(dir string, name string, size int) font.Face {
	key := fontKey{name: name, size: size}
	if v, ok := c.cache.Load(key); ok {
		return v.(font.Face)
	}
	fontFace := loadFont(dir, name, size)
	c.cache.Store(key, fontFace)
	return fontFace
}

// Helper interface to allow passing nil in old code paths if needed, though we will update them.
type FontLoader interface {
	LoadFont(dir string, name string, size int) font.Face
}

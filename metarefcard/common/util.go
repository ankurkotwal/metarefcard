package common

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"gopkg.in/yaml.v3"
)

// MockSet is a map masquerading as a set
type MockSet map[string]string

// Keys returns a MockSet as an array
func (m MockSet) Keys() []string {
	array := make([]string, 0, len(m))
	for k := range m {
		array = append(array, k)
	}
	return array
}

// LoadYaml loads Yaml file and prints any errors
func LoadYaml(filename string, out interface{}, label string, log *Logger) {
	yamlData, err := ioutil.ReadFile(filename)
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

// LoadFont loads a font into memory and returns it.
func LoadFont(dir string, name string, size int) font.Face {
	fontPath := fmt.Sprintf("%s/%s", dir, name)

	fontBytes, err := ioutil.ReadFile(fontPath)
	if err != nil {
		panic(err)
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		panic(err)
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size: float64(size),
	})
	return face
}

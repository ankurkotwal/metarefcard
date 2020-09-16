package util

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"gopkg.in/yaml.v3"
)

// LoadFont returns a font as per size
func LoadFont(filename string, fontSize float64) font.Face {
	font, err := gg.LoadFontFace(filename, fontSize)
	if err != nil {
		panic(err)
	}
	return font
}

// LoadYaml loads Yaml file and prints any errors
func LoadYaml(filename string, out interface{}, debugOutput bool, debugLabel string) {
	yamlData, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("error: yaml ioutil.ReadFile %v ", err)
	}
	err = yaml.Unmarshal([]byte(yamlData), out)
	if err != nil {
		log.Fatalf("error: yaml.Unmarshal %v", err)
	}
	if debugOutput {
		PrintYamlObject(out, debugLabel)
	}
}

// PrintYamlObject outputs contents of yaml object with a label
func PrintYamlObject(in interface{}, label string) {
	d, err := yaml.Marshal(in)
	if err != nil {
		log.Fatalf("error: yaml.Marshal %v", err)
	}
	fmt.Printf("=== %s ===\n%s\n\n", label, string(d))

}

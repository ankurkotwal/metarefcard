package util

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

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

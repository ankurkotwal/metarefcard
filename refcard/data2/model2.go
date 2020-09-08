package data2

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

// Example
// x55 -> { image, devices: { stick: { inputs: { button1: {isDigital, x, y, width, height}}}}}

// DeviceModel -
type DeviceModel map[string]map[string]struct {
	Image       string `yaml:"Image"`
	DisplayName string `yaml:"DisplayName"`
	Inputs      map[string]struct {
		IsDigital bool `yaml:"IsDigital"`
		ImageX    int  `yaml:"OffsetX"`
		ImageY    int  `yaml:"OffsetY"`
		Width     int  `yaml:"Width"`
		Height    int  `yaml:"Height"`
	} `yaml:"Inputs"`
}

// T - blah
type T DeviceModel

// LoadDeviceData - Reads device data from files
func LoadDeviceData() {
	t := T{}

	yamlData, err := ioutil.ReadFile("data/generatedDevices.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal([]byte(yamlData), &t)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- t:\n%v\n\n", t)

	d, err := yaml.Marshal(&t)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- t dump:\n%s\n\n", string(d))

	m := make(map[interface{}]interface{})

	err = yaml.Unmarshal([]byte(yamlData), &m)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- m:\n%v\n\n", m)

	d, err = yaml.Marshal(&m)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- m dump:\n%s\n\n", string(d))
}

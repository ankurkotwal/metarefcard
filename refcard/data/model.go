package data

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

// Example
// x55 -> { image, devices: { stick: { inputs: { button1: {isDigital, x, y, width, height}}}}}

// DeviceMap - structure of devices (by group name)
type DeviceMap map[string]struct {
	Image   string `yaml:"Image"`
	Devices map[string]struct {
		DisplayName string `yaml:"DisplayName"`
		Inputs      map[string]struct {
			IsDigital bool `yaml:"IsDigital"`
			ImageX    int  `yaml:"OffsetX"`
			ImageY    int  `yaml:"OffsetY"`
			Width     int  `yaml:"Width"`
			Height    int  `yaml:"Height"`
		} `yaml:"Inputs"`
	} `yaml:"Devices"`
}

// LoadDeviceData - Reads device data from files
func LoadDeviceData(neededDeviceGroups *map[string]bool, debugOutput bool) *DeviceMap {
	deviceMap := DeviceMap{}
	yamlData, err := ioutil.ReadFile("data/generatedDevices.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal([]byte(yamlData), &deviceMap)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if debugOutput {
		d, err := yaml.Marshal(&deviceMap)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("=== Full Device Map ===\n%s\n\n", string(d))
	}

	// Filter for only the device groups we're interested in
	for groupName, groupData := range deviceMap {
		foundDevice := false
		for shortName := range groupData.Devices {
			if (*neededDeviceGroups)[shortName] {
				foundDevice = true
			}
		}
		if !foundDevice {
			delete(deviceMap, groupName)
		}
	}

	if debugOutput {
		d, err := yaml.Marshal(&deviceMap)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("=== Targeted Device Map ===\n%s\n\n", string(d))
	}
	return &deviceMap
}

// TODO - Keyboard handling

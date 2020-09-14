package data

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

// Example
// x55 -> { image, devices: { stick: { inputs: { button1: {isDigital, x, y, width, height}}}}}

// deviceMap - structure of devices (by group name)
type deviceMap map[string]struct {
	Image   string                     `yaml:"Image"`
	Devices map[string]DeviceInputData `yaml:"Devices"`
}

// DeviceInputData - data about a given device
type DeviceInputData struct {
	DisplayName string    `yaml:"DisplayName"`
	Inputs      InputsMap `yaml:"Inputs"`
}

// InputData - data relating to a given input
type InputData struct {
	IsDigital bool `yaml:"IsDigital"`
	ImageX    int  `yaml:"OffsetX"`
	ImageY    int  `yaml:"OffsetY"`
	Width     int  `yaml:"Width"`
	Height    int  `yaml:"Height"`
}

// DeviceModel - structure to store image and inputs
type DeviceModel map[string]*DeviceData

// DeviceData - information about a device
type DeviceData struct {
	Image  string
	Inputs InputsMap
}

// InputsMap - Map of input data by name
type InputsMap map[string]InputData

// OverlaysByImage - image overlay data indexed by image name
// Image -> Input -> OverlayData
type OverlaysByImage map[string]map[string]OverlayData

// OverlayData - data about what to put in overlay, grouping and location
type OverlayData struct {
	Context    string
	Text       string
	PosAndSize *InputData
}

// LoadDeviceData - Reads device data from files
func LoadDeviceData(neededDevices map[string]bool, debugOutput bool) DeviceModel {
	deviceMap := deviceMap{}
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

	deviceModel := make(DeviceModel)
	// Filter for only the device groups we're interested in
	for _, groupData := range deviceMap {
		for shortName, inputData := range groupData.Devices {
			if neededDevices[shortName] {
				deviceData := new(DeviceData)
				deviceModel[shortName] = deviceData
				deviceData.Image = groupData.Image
				deviceData.Inputs = inputData.Inputs
			}
		}
	}

	if debugOutput {
		d, err := yaml.Marshal(&deviceModel)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("=== Targeted Device Map ===\n%s\n\n", string(d))
	}
	return deviceModel
}

// TODO - Keyboard handling

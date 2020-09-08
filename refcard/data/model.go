package data

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

// Example
// x55 -> { image, devices: { stick: { inputs: { button1: {isDigital, x, y, width, height}}}}}

// DeviceToGroup - map of device name to the group it belongs to. Group is the key in DeviceIndexByGroupName
type DeviceToGroup map[string]string

// DeviceIndexByGroupName - master index, keyed by group name
type DeviceIndexByGroupName map[string]*DeviceGroup

// DeviceGroup - image and map of devices by name
type DeviceGroup struct {
	Image   string
	Devices DevicesByName
}

// DevicesByName - map of devices by name
type DevicesByName map[string]*DeviceData

// DeviceData - information about a device
type DeviceData struct {
	// DeviceData - data about a specific device
	DisplayName     string
	InputDataByName *InputDataByName
}

// InputDataByName - map of device input data by input name
type InputDataByName map[string]*InputData

// InputData - data about the input & image location
type InputData struct {
	IsDigital   bool
	ImageX      int
	ImageY      int
	ImageWidth  int
	ImageHeight int
}

// TODO - Keyboard handling

// PrintDeviceIndex - prints the full device index
func PrintDeviceIndex(deviceIndex *DeviceIndexByGroupName) {
	for groupName, deviceGroup := range *deviceIndex {
		for deviceName, deviceData := range deviceGroup.Devices {
			for inputName, data := range *(deviceData.InputDataByName) {
				fmt.Printf("%s %s %s %s %s %t %d %d %d %d\n",
					groupName, deviceGroup.Image, deviceName, deviceData.DisplayName, inputName,
					data.IsDigital, data.ImageX, data.ImageY, data.ImageWidth, data.ImageHeight)
			}
		}
	}
}

// T - blah
type T DeviceIndexByGroupName

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

	// d, err = yaml.Marshal(&m)
	// if err != nil {
	// 	log.Fatalf("error: %v", err)
	// }
	// fmt.Printf("--- m dump:\n%s\n\n", string(d))
}

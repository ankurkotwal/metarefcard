package data

import "fmt"

// Example
// x55 -> { image, devices: { stick: { inputs: { button1: {isDigital, x, y, width, height}}}}}

// DeviceIndexByGroupName - master index, keyed by group name
type DeviceIndexByGroupName map[string]*DeviceGroup

// DeviceGroup - image and map of devices by name
type DeviceGroup struct {
	Image   string
	Devices DevicesByName
}

// DevicesByName - map of devices by name
type DevicesByName map[string]*DeviceData

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
func PrintDeviceIndex(deviceIndex DeviceIndexByGroupName) {
	for groupName, deviceGroup := range deviceIndex {
		for deviceName, deviceData := range deviceGroup.Devices {
			for inputName, data := range *(deviceData.InputDataByName) {
				fmt.Printf("%s %s %s %s %s %t %d %d %d %d\n",
					groupName, deviceGroup.Image, deviceName, deviceData.DisplayName, inputName,
					data.IsDigital, data.ImageX, data.ImageY, data.ImageWidth, data.ImageHeight)
			}
		}
	}
}

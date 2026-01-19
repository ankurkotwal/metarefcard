package common

// LoadDevicesInfo loads all the device information (across files) into "devices"
func LoadDevicesInfo(file string, devices *Devices, log *Logger) {
	if err := LoadYaml(file, devices); err != nil {
		log.Fatal("LoadDevicesInfo LoadYaml %v", err)
	}

	var generatedDevices GeneratedDevices
	if err := LoadYaml(devices.GeneratedFile, &generatedDevices); err != nil {
		log.Fatal("LoadDevicesInfo GeneratedFile LoadYaml %v", err)
	}

	// Add device additions to the main device index
	for shortName, inputs := range devices.Index {
		generatedInputs, found := generatedDevices.Index[shortName]
		if !found {
			generatedInputs = make(DeviceInputs)
			generatedDevices.Index[shortName] = generatedInputs
		}

		// Already have some inputs. Need to override one at a time
		for input, additionalInput := range inputs {
			generatedInputs[input] = additionalInput
		}
	}
	devices.Index = generatedDevices.Index

	// Add image map additions
	for shortName, image := range devices.ImageMap {
		generatedDevices.ImageMap[shortName] = image
	}
	devices.ImageMap = generatedDevices.ImageMap
}

// Devices holds all the device related data
type Devices struct {
	GeneratedFile        string                  `yaml:"GeneratedFile"`
	Index                DeviceMap               `yaml:"DeviceMap"`
	ImageMap             ImageMap                `yaml:"ImageMap"`
	DeviceToShortNameMap DeviceNameFullToShort   `yaml:"DeviceNameMap"`
	DeviceLabelsByImage  map[string]string       `yaml:"DeviceLabelsByImage"`
	ImageSizeOverride    map[string]Dimensions2d `yaml:"ImageSizeOverride"` // Device Name -> Dimensions2d
}

// GeneratedDevices holds structure from the generated config
type GeneratedDevices struct {
	Index    DeviceMap `yaml:"DeviceMap"`
	ImageMap ImageMap  `yaml:"ImageMap"`
}

// DeviceMap - structure of devices. short name -> inputs
type DeviceMap map[string]DeviceInputs

// DeviceInputs - structure of inputs for a device. input -> input data
type DeviceInputs map[string]InputData

// InputData - data relating to a given input
type InputData struct {
	X int `yaml:"x"` // X location
	Y int `yaml:"y"` // Y location
	W int `yaml:"w"` // Width
	H int `yaml:"h"` // Height
}

// ImageMap - contains device short name -> image name
type ImageMap map[string]string

// DeviceNameFullToShort maps game device full names to MetaRefCard short names
type DeviceNameFullToShort map[string]string

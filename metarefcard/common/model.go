package common

import (
	"regexp"
	"sort"
)

// LoadGameModel - load game specific data from our model. Update the device names
// (map game device name to our model names)
func LoadGameModel(filename string, label string, debugOutput bool) *GameData {
	data := GameData{}
	LoadYaml(filename, &data, label)

	fullToShort := DeviceNameFullToShort{}
	// Update map of game device names to our model device names
	for fullName, shortName := range data.DeviceNameMap {
		if shortName != "" {
			fullToShort[fullName] = shortName
		} else {
			fullToShort[fullName] = DeviceMissingInfo
		}
	}
	data.DeviceNameMap = fullToShort

	return &data
}

// OrgDeviceModel - Returns only the devices that the caller is asking for
func OrgDeviceModel(deviceMap DeviceMap, neededDevices MockSet, config *Config) DeviceModel {
	deviceModel := make(DeviceModel)
	// Filter for only the device groups we're interested in
	for _, groupData := range deviceMap {
		for shortName, inputData := range groupData.Devices {
			if _, found := neededDevices[shortName]; found {
				deviceData := new(DeviceData)
				deviceModel[shortName] = deviceData
				deviceData.Image = groupData.Image
				deviceData.Inputs = inputData.Inputs
			}
		}
	}

	// Add device additions to the main device index
	for deviceName, deviceInputData := range config.InputOverrides {
		if deviceData, found := deviceModel[deviceName]; found {
			for additionInput, additionData := range deviceInputData.Inputs {
				deviceData.Inputs[additionInput] = additionData
			}
		}
	}

	if config.DebugOutput {
		PrintYamlObject(&deviceModel, "Targeted Device Map")
	}
	return deviceModel
}

// GenerateContextColours - basic utility function to generate colours
func GenerateContextColours(contexts MockSet, config *Config) {
	contextKeys := contexts.Keys()
	sort.Strings(contextKeys)
	i := 0
	for _, context := range contextKeys {
		if i >= len(config.AlternateColours) {
			// Ran out of colours, repeat
			i = 0
		}
		// Only move to next colour if this is an unseen category
		contexts[context] = config.AlternateColours[i]
		i++
	}
}

// RegexByName - map of named regex strings
type RegexByName map[string]*regexp.Regexp

// DeviceNameToImage - contains device short name -> image name
type DeviceNameToImage map[string]string

// DeviceMap - structure of devices (by group name)
type DeviceMap map[string]struct {
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

// DeviceModel - structure to store image and inputs, keyed by device shortname
type DeviceModel map[string]*DeviceData

// DeviceData - information about a device
type DeviceData struct {
	Image  string
	Inputs InputsMap
}

// InputsMap - Map of input data by name
type InputsMap map[string]InputData

// OverlaysByImage - image overlay data indexed by image name
// Image -> Device:Input -> OverlayData
type OverlaysByImage map[string]map[string]*OverlayData

// OverlayData - data about what to put in overlay, grouping and location
type OverlayData struct {
	ContextToTexts map[string][]string
	PosAndSize     *InputData
}

// GameData holds the game's parsed data
type GameData struct {
	DeviceNameMap DeviceNameFullToShort  `yaml:"DeviceNameMap"`
	InputMap      DeviceInputTypeMapping `yaml:"InputMapping"`
	InputLabels   map[string]string      `yaml:"InputLabels"`
	Regexes       map[string]string      `yaml:"Regexes"`
}

// DeviceNameFullToShort maps game device full names to MetaRefCard short names
type DeviceNameFullToShort map[string]string

// DeviceInputTypeMapping contains a map of device short names to
// types of input maps (e.g. Slider, Axis, Rotaion)
type DeviceInputTypeMapping map[string]InputTypeMapping

// InputTypeMapping maps the type of input (e.g Slider, Axis, Rotation) to
// a map of game input to MetaRefCard input (e.g. X-Axis -> U-Axis)
type InputTypeMapping map[string]map[string]string // Device -> Type (Axis/Slider) -> axisInputMap/sliderInputMap

const (
	// DeviceUnknown - unfamiliar with this device
	DeviceUnknown = "DeviceUnknown"
	// DeviceMissingInfo - only know the name of device
	DeviceMissingInfo = "DeviceMissingInfo"
)

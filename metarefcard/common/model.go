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

// FilterDevices - Returns only the devices that the caller is asking for
func FilterDevices(deviceMap DeviceMap, neededDevices map[string]bool, debugOutput bool) DeviceModel {
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
		PrintYamlObject(&deviceModel, "Targeted Device Map")
	}
	return deviceModel
}

// GenerateContextColours - basic utility function to generate colours
func GenerateContextColours(contexts []string, config *Config) map[string]string {
	sort.Strings(contexts)
	categories := make(map[string]string) // Context -> Colour
	i := 0
	for _, category := range contexts {
		if i >= len(config.AlternateColours) {
			// Ran out of colours, repeat
			i = 0
		}
		if _, found := categories[category]; !found {
			// Only move to next colour if this is an unseen category
			categories[category] = config.AlternateColours[i]
			i++
		}
	}
	return categories
}

// RegexByName - map of named regex strings
type RegexByName map[string]*regexp.Regexp

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
// Image -> Device:Input -> OverlayData
type OverlaysByImage map[string]map[string]*OverlayData

// OverlayData - data about what to put in overlay, grouping and location
type OverlayData struct {
	ContextToTexts map[string][]string
	PosAndSize     *InputData
}

// GameData holds the game's parsed data
type GameData struct {
	DeviceNameMap  DeviceNameFullToShort      `yaml:"DeviceNameMap"`
	InputMap       DeviceInputTypeMapping     `yaml:"InputMapping"`
	InputOverrides map[string]DeviceInputData `yaml:"InputOverrides"`
	InputLabels    map[string]string          `yaml:"InputLabels"`
	Regexes        map[string]string          `yaml:"Regexes"`
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
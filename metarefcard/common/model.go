package common

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

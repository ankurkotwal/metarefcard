package data

// Example
// x55 -> { image, devices: { stick: { inputs: { button1: {isDigital, x, y, width, height}}}}}

// DeviceIndexByGroupName - master index, keyed by group name
type DeviceIndexByGroupName map[string]DeviceGroup

// DeviceGroup - image and map of devices by name
type DeviceGroup struct {
	Image   string
	Devices DevicesByName
}

// DevicesByName - map of devices by name
type DevicesByName map[string]DeviceInputDataByInputName

// DeviceInputDataByInputName - map of device input data by input name
type DeviceInputDataByInputName map[string]DeviceInputData

// DeviceInputData - data about the input & image location
type DeviceInputData struct {
	IsDigital   bool
	ImageX      int
	ImageY      int
	ImageWidth  int
	ImageHeight int
}

// TODO - Things we need
// Keyboard handling

package common

import "regexp"

// Config contains all the configuration data for the app
type Config struct {
	AppName       string `yaml:"AppName"`
	Version       string `yaml:"Version"`
	DebugOutput   bool   `yaml:"DebugOutput"`
	VerboseOutput bool   `yaml:"VerboseOutput"`

	DevicesFile    string    `yaml:"DevicesModel"`
	DeviceMap      DeviceMap `yaml:"DeviceMap"`
	InputOverrides DeviceMap `yaml:"InputOverrides"`

	ImageMap          ImageMap                `yaml:"ImageMap"`
	DefaultImage      Dimensions2d            `yaml:"DefaultImage"`
	PixelMultiplier   float64                 `yaml:"PixelMultiplier"`
	ImagesDir         string                  `yaml:"ImagesDir"`
	ImageSizeOverride map[string]Dimensions2d `yaml:"ImageSizeOverride"` // Device Name -> Dimensions2d

	FontsDir          string  `yaml:"FontsDir"`
	InputFont         string  `yaml:"InputFont"`
	InputFontSize     float64 `yaml:"InputFontSize"`
	DefaultLineHeight int     `yaml:"DefaultLineHeight"`
	InputPixelInset   int     `yaml:"InputPixelInset"`

	BackgroundColour string   `yaml:"BackgroundColour"`
	LightColour      string   `yaml:"LightColour"`
	DarkColour       string   `yaml:"DarkColour"`
	AlternateColours []string `yaml:"AlternateColours"`
}

// GeneratedConfig holds structure from the generated config
type GeneratedConfig struct {
	DeviceMap DeviceMap `yaml:"DeviceMap"`
	ImageMap  ImageMap  `yaml:"ImageMap"`
}

// Dimensions2d contains width and height
type Dimensions2d struct {
	W int `yaml:"w"` // Width
	H int `yaml:"h"` // Height
}

// GameData holds the game's parsed data
type GameData struct {
	DeviceNameMap DeviceNameFullToShort  `yaml:"DeviceNameMap"`
	InputMap      DeviceInputTypeMapping `yaml:"InputMap"`
	InputLabels   map[string]string      `yaml:"InputLabels"`
	Regexes       map[string]string      `yaml:"Regexes"`
}

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

// RegexByName - map of named regex strings
type RegexByName map[string]*regexp.Regexp

// ImageMap - contains device short name -> image name
type ImageMap map[string]string

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

// OverlaysByImage - image overlay data indexed by image name
// Image -> Device:Input -> OverlayData
type OverlaysByImage map[string]map[string]*OverlayData

// OverlayData - data about what to put in overlay, grouping and location
type OverlayData struct {
	ContextToTexts map[string][]string
	PosAndSize     *InputData
}

// DeviceNameFullToShort maps game device full names to MetaRefCard short names
type DeviceNameFullToShort map[string]string

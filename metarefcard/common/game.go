package common

import "regexp"

// GameData holds the game's parsed data
type GameData struct {
	Logo        string                 `yaml:"Logo"`
	Regexes     map[string]string      `yaml:"Regexes"`
	InputMap    DeviceInputTypeMapping `yaml:"InputMap"`
	InputLabels map[string]string      `yaml:"InputLabels"`
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

// OverlaysByImage - image overlay data indexed by image name
// Image -> Device:Input -> OverlayData
type OverlaysByImage map[string]map[string]*OverlayData

// OverlayData - data about what to put in overlay, grouping and location
type OverlayData struct {
	ContextToTexts map[string][]string
	PosAndSize     *InputData
}

// GameBindsByDevice - Short name -> Context -> Action -> Primary/Secondary -> Key
type GameBindsByDevice map[string]GameContextActions

// GameContextActions - Context -> Action
type GameContextActions map[string]GameActions

// GameActions - Action -> Input
type GameActions map[string]GameInput

// GameInput - Array of inputs. Index of InputPrimary and InputSecondary
type GameInput []string

const (
	// InputPrimary - primary input
	InputPrimary = 0
	// InputSecondary - secondary input
	InputSecondary = 1
	// NumInputs - number of inputs.
	NumInputs = 2
)

package common

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

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
	// ProfileDefault - the name of the default profile
	ProfileDefault = "default_metarefcard"
)

// Caser that returns Title case for a string.
var titleCaser = cases.Title(language.AmericanEnglish)

func TitleCaser(text string) string {
	return titleCaser.String(text)
}

// RegexByName - map of named regex strings
type RegexByName map[string]*regexp.Regexp

// OverlaysByProfile - image overlay object indexed by profile
// Profile -> OverlaysByImage
type OverlaysByProfile map[string]OverlaysByImage

// OverlaysByImage - image overlay data indexed by image name
// Image -> Device:Input -> OverlayData
type OverlaysByImage map[string]map[string]OverlayData

// OverlayData - data about what to put in overlay, grouping and location
type OverlayData struct {
	ContextToTexts map[string][]string
	PosAndSize     InputData
}

// GameBindsByProfile - Profile -> Short name -> Context -> Action -> Primary/Secondary -> Key
type GameBindsByProfile map[string]GameDeviceContextActions

// GameDeviceContextActions - Short name -> Context -> Action -> Primary/Secondary -> Key
type GameDeviceContextActions map[string]GameContextActions

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

// GameBindsAsString returns the object as a printable string
func GameBindsAsString(gameBindsByProfile GameBindsByProfile) string {
	info := make([]string, 0)
	info = append(info, "=== Loaded FS2020 Config ===\n")
	for profile, gameBinds := range gameBindsByProfile {
		info = append(info, fmt.Sprintf("Profile=\"%s\"", profile))
		for shortName, gameDevice := range gameBinds {
			info = append(info, fmt.Sprintf("  DeviceName=\"%s\"", shortName))
			for contextName, actions := range gameDevice {
				info = append(info, fmt.Sprintf("    ContextName=\"%s\"\n", contextName))
				for actionName, action := range actions {
					secondaryText := ""
					if len(action[InputSecondary]) != 0 {
						secondaryText = fmt.Sprintf("   SecondaryInfo=\"%s\"",
							action[InputSecondary])
					}
					info = append(info, fmt.Sprintf("      ActionName=\"%s\" PrimaryInfo=\"%s\" %s\n",
						actionName, action[InputPrimary], secondaryText))
				}
			}
		}
	}
	return strings.Join(info, "")
}

// ContextToColours is a mapping of game contexts to colours that are used for visual grouping
type ContextToColours map[string]string

// Keys returns a MockSet as an array
func (m ContextToColours) Keys() []string {
	array := make([]string, 0, len(m))
	for k := range m {
		array = append(array, k)
	}
	return array
}

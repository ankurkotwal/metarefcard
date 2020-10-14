package fs2020

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/ankurkotwal/MetaRefCard/metarefcard/common"
)

var initiliased bool = false
var sharedRegexes fs2020Regexes
var sharedGameData *common.GameData

// HandleRequest services the request to load files
func HandleRequest(files [][]byte, config *common.Config) (*common.GameData,
	common.GameBindsByDevice, common.MockSet, common.MockSet, string) {
	if !initiliased {
		sharedGameData = common.LoadGameModel("config/fs2020.yaml",
			"FS2020 Data", config.DebugOutput)
		sharedRegexes.Button = regexp.MustCompile(sharedGameData.Regexes["Button"])
		sharedRegexes.Axis = regexp.MustCompile(sharedGameData.Regexes["Axis"])
		sharedRegexes.Pov = regexp.MustCompile(sharedGameData.Regexes["Pov"])
		sharedRegexes.Rotation = regexp.MustCompile(sharedGameData.Regexes["Rotation"])
		sharedRegexes.Slider = regexp.MustCompile(sharedGameData.Regexes["Slider"])
		initiliased = true
	}
	gameBinds, gameDevices, gameContexts := loadInputFiles(files, sharedGameData.DeviceNameMap,
		config.DebugOutput, config.VerboseOutput)
	common.GenerateContextColours(gameContexts, config)
	return sharedGameData, gameBinds, gameDevices, gameContexts, sharedGameData.Logo
}

// Load the game config files (provided by user)
func loadInputFiles(files [][]byte, deviceShortNameMap common.DeviceNameFullToShort,
	debugOutput bool, verboseOutput bool) (common.GameBindsByDevice, common.MockSet, common.MockSet) {

	gameBinds := make(common.GameBindsByDevice)
	neededDevices := make(common.MockSet)
	contexts := make(common.MockSet)

	// XML state variables
	var currentContext common.GameActions
	var contextActions common.GameContextActions
	currentAction := make(common.GameInput, common.NumInputs)
	currentKeyType := keyUnknown
	var currentKey *int

	for idx, file := range files {
		_ = idx
		decoder := xml.NewDecoder(bytes.NewReader(file))
		for {
			token, err := decoder.Token()
			if token == nil || err == io.EOF {
				// EOF means we're done.
				break
			} else if err != nil {
				log.Fatalf("Error: FS2020 decoding token: %s", err)
			}

			switch ty := token.(type) {
			case xml.StartElement:
				switch ty.Name.Local {
				case "Device":
					// Found new device
					var aDevice string
					for _, attr := range ty.Attr {
						switch attr.Name.Local {
						case "DeviceName":
							aDevice = attr.Value
							break
						}
					}
					var found bool
					var shortName string
					if shortName, found = deviceShortNameMap[aDevice]; !found {
						log.Printf("Error FS2020 could not find short name for %s\n",
							aDevice)
						break // Move on to next device
					}
					_, found = gameBinds[shortName]
					if found {
						out, _ := json.Marshal(aDevice)
						log.Printf("Error: FS2020 duplicate device: %s\n", out)
						break // Move on to next device
					} else {
						if debugOutput {
							out, _ := json.Marshal(aDevice)
							log.Printf("Info: FS2020 new device: %s\n", out)
						}
						contextActions = make(common.GameContextActions)
						neededDevices[shortName] = "" // Add to set
						gameBinds[shortName] = contextActions
						if shortName == common.DeviceMissingInfo {
							log.Printf("Error: FS2020 missing info for device '%s'\n",
								shortName)
						}
					}
				case "Context":
					// Found new context
					var contextName string
					for _, attr := range ty.Attr {
						if attr.Name.Local == "ContextName" {
							contextName = attr.Value
							contexts[contextName] = ""
							break
						}
					}
					var found bool
					currentContext, found = contextActions[contextName]
					if found {
						log.Printf("Error: FS2020 duplicate context: %s\n", contextName)
					} else {
						if debugOutput {
							log.Printf("FS2020 new context: %s\n", contextName)
						}
						currentContext = make(common.GameActions)
						contextActions[contextName] = currentContext
					}
				case "Action":
					// Found new action
					currentKeyType = keyUnknown
					var actionName string
					action := make(common.GameInput, common.NumInputs)
					for _, attr := range ty.Attr {
						if attr.Name.Local == "ActionName" {
							actionName = attr.Value
						}
					}
					var found bool
					currentAction, found = currentContext[actionName]
					if found {
						log.Printf("Error: FS2020 duplicate action: %s\n", actionName)
					} else {
						if debugOutput {
							log.Printf("FS2020 new action: %s\n", actionName)
						}
						currentAction = action
						currentContext[actionName] = currentAction
					}
				case "Primary":
					currentKeyType = keyPrimary
				case "Secondary":
					currentKeyType = keySecondary
				case "KEY":
					for _, attr := range ty.Attr {
						if attr.Name.Local == "Information" {
							switch currentKeyType {
							case keyPrimary:
								currentAction[common.InputPrimary] = attr.Value
							case keySecondary:
								currentAction[common.InputSecondary] = attr.Value
							}
						}
						break
					}
				}
			case xml.CharData:
				if currentKey != nil {
					value := string([]byte(ty))
					*currentKey, err = strconv.Atoi(value)
					if err != nil {
						log.Printf("Error: FS2020 primary key value %s parsing error\n", value)
					}
				}
			case xml.EndElement:
				switch ty.Name.Local {
				case "Device":
				case "Context":
					currentContext = nil
				case "Action":
					currentAction = nil
				case "Primary":
					currentKeyType = keyUnknown
				case "Secondary":
					currentKeyType = keyUnknown
				case "KEY":
					currentKey = nil
				}
			}
		}
	}

	if verboseOutput {
		log.Printf("=== Loaded FS2020 Config ===\n")
		for shortName, gameDevice := range gameBinds {
			log.Printf("DeviceName=\"%s\"", shortName)
			for contextName, actions := range gameDevice {
				log.Printf("  ContextName=\"%s\"\n", contextName)
				for actionName, action := range actions {
					secondaryText := ""
					if len(action[common.InputSecondary]) != 0 {
						secondaryText = fmt.Sprintf(" SecondaryInfo=\"%s\"",
							action[common.InputSecondary])
					}
					log.Printf("    ActionName=\"%s\" PrimaryInfo=\"%s\" %s\n",
						actionName, action[common.InputPrimary], secondaryText)
				}
			}
		}
	}

	return gameBinds, neededDevices, contexts
}

// MatchGameInputToModel takes the game provided bindings with the device map to
// build a list of image overlays.
func MatchGameInputToModel(deviceName string, actionData common.GameInput,
	deviceInputs common.DeviceInputs, gameInputMap common.InputTypeMapping) (common.GameInput, string) {
	inputLookups := make([]string, 0, 2)

	// First the primary input for this action
	input := matchGameInputToModelByRegex(deviceName, actionData[common.InputPrimary],
		deviceInputs, gameInputMap)
	if input != "" {
		inputLookups = append(inputLookups, input)
	} else {
		log.Printf("Error: FS2020 did not find primary input for %s\n", actionData[common.InputPrimary])
	}
	// Now the secondary input
	if len(actionData[common.InputSecondary]) > 0 {
		input := matchGameInputToModelByRegex(deviceName, actionData[common.InputSecondary],
			deviceInputs, gameInputMap)
		if input != "" {
			inputLookups = append(inputLookups, input)
		} else {
			log.Printf("Error: FS2020 did not find secondary input for %s\n",
				actionData[common.InputSecondary])
		}
	}
	return inputLookups, sharedGameData.Logo
}

// Matches an action to a device's inputs using regexes. Returns string to lookup input
func matchGameInputToModelByRegex(deviceName string, action string,
	inputs common.DeviceInputs, gameInputMap common.InputTypeMapping) string {
	var matches [][]string

	matches = sharedRegexes.Button.FindAllStringSubmatch(action, -1)
	if matches != nil {
		return matches[0][1]
	}

	matches = sharedRegexes.Axis.FindAllStringSubmatch(action, -1)
	if matches != nil {
		axis := fmt.Sprintf("%s%s", matches[0][1], matches[0][2])
		if gameInputMap != nil {
			if subAxis, found := gameInputMap["Axis"]; found {
				if substituteAxis, found := subAxis[axis]; found {
					axis = substituteAxis
				}
			}
		}
		axis = fmt.Sprintf("%sAxis", axis)
		return axis
	}
	matches = sharedRegexes.Pov.FindAllStringSubmatch(action, -1)
	if matches != nil {
		direction := strings.Title(strings.ToLower(matches[0][2]))
		pov := fmt.Sprintf("POV%s%s", "1", direction)
		if len(matches[0][1]) > 0 {
			pov = fmt.Sprintf("POV%s%s", matches[0][1], direction)
		}
		return pov
	}

	matches = sharedRegexes.Rotation.FindAllStringSubmatch(action, -1)
	if matches != nil {
		rotation := fmt.Sprintf("R%sAxis", matches[0][1])
		if input, ok := gameInputMap["Rotation"]; ok {
			// Check override
			rotation = fmt.Sprintf("%sAxis", input[matches[0][1]])
		}
		return rotation
	}

	matches = sharedRegexes.Slider.FindAllStringSubmatch(action, -1)
	if matches != nil {
		var slider string
		if input, ok := gameInputMap["Slider"]; ok {
			slider = fmt.Sprintf("%sAxis", input[matches[0][1]])
		} else {
			log.Printf("Error: FS2020 unknown action %s for slider on device %s\n", action, deviceName)
			return ""
		}
		if _, ok := inputs[slider]; ok {
			return slider
		}
		log.Printf("Error: FS2020 couldn't find slider %s on device %s\n", slider, deviceName)
		return ""

	}
	log.Printf("Error: FS2020 could not find matching Action %s on device %s\n", action, deviceName)
	return ""
}

const (
	keyUnknown   = iota
	keyPrimary   = iota
	keySecondary = iota
)

type fs2020Regexes struct {
	Button   *regexp.Regexp
	Axis     *regexp.Regexp
	Pov      *regexp.Regexp
	Rotation *regexp.Regexp
	Slider   *regexp.Regexp
}

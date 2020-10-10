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
var gameData *common.GameData
var regexes common.RegexByName

// HandleRequest services the request to load files
func HandleRequest(files [][]byte,
	config *common.Config) (common.OverlaysByImage, map[string]string) {
	if !initiliased {
		gameData = common.LoadGameModel("config/fs2020.yaml",
			"FS2020 Data", config.DebugOutput)
		regexes = make(common.RegexByName)
		// TODO perf - make regexes a struct instead of a map
		for name, pattern := range gameData.Regexes {
			regexes[name] = regexp.MustCompile(pattern)
		}
		initiliased = true
	}
	gameBinds, neededDevices, contexts := loadInputFiles(files, gameData.DeviceNameMap,
		config.DebugOutput, config.VerboseOutput)
	common.GenerateContextColours(contexts, config)
	deviceMap := common.FilterDevices(neededDevices, config)
	return populateImageOverlays(deviceMap, config.ImageMap, gameBinds, gameData), contexts
}

// Load the game config files (provided by user)
func loadInputFiles(files [][]byte, deviceShortNameMap common.DeviceNameFullToShort,
	debugOutput bool, verboseOutput bool) (fs2020BindsByDevice, common.MockSet, common.MockSet) {

	gameBinds := make(fs2020BindsByDevice)
	neededDevices := make(common.MockSet)
	contexts := make(common.MockSet)

	// XML state variables
	var currentDevice *fs2020Device
	var currentContext map[string]*fs2020Input
	var currentAction *fs2020Input
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
					var aDevice fs2020Device
					for _, attr := range ty.Attr {
						switch attr.Name.Local {
						case "DeviceName":
							aDevice.DeviceFullName = attr.Value
							break
						}
					}
					var found bool
					var shortName string
					if shortName, found = deviceShortNameMap[aDevice.DeviceFullName]; !found {
						log.Printf("Error FS2020 could not find short name for %s\n",
							aDevice.DeviceFullName)
						break // Move on to next device
					}
					currentDevice, found = gameBinds[shortName]
					if found {
						out, _ := json.Marshal(aDevice)
						log.Printf("Error: FS2020 duplicate device: %s\n", out)
						break // Move on to next device
					} else {
						if debugOutput {
							out, _ := json.Marshal(aDevice)
							log.Printf("Info: FS2020 new device: %s\n", out)
						}
						currentDevice = &aDevice
						currentDevice.ContextActions = make(fs2020ContextActions)
						neededDevices[shortName] = "" // Add to set
						gameBinds[shortName] = currentDevice
						if shortName == common.DeviceMissingInfo {
							log.Printf("Error: FS2020 missing info for device '%s'\n",
								aDevice.DeviceFullName)
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
					currentContext, found = currentDevice.ContextActions[contextName]
					if found {
						log.Printf("Error: FS2020 duplicate context: %s\n", contextName)
					} else {
						if debugOutput {
							log.Printf("FS2020 new context: %s\n", contextName)
						}
						currentContext = make(map[string]*fs2020Input)
						currentDevice.ContextActions[contextName] = currentContext
					}
				case "Action":
					// Found new action
					currentKeyType = keyUnknown
					var actionName string
					var action fs2020Input
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
						currentAction = &action
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
								currentAction.PrimaryInput = attr.Value
							case keySecondary:
								currentAction.SecondaryInput = attr.Value
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
					currentDevice = nil
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
		for _, gameDevice := range gameBinds {
			log.Printf("DeviceName=\"%s\"", gameDevice.DeviceFullName)
			for contextName, actions := range gameDevice.ContextActions {
				log.Printf("  ContextName=\"%s\"\n", contextName)
				for actionName, action := range actions {
					secondaryText := ""
					if len(action.SecondaryInput) != 0 {
						secondaryText = fmt.Sprintf(" SecondaryInfo=\"%s\"",
							action.SecondaryInput)
					}
					log.Printf("    ActionName=\"%s\" PrimaryInfo=\"%s\" %s\n",
						actionName, action.PrimaryInput, secondaryText)
				}
			}
		}
	}

	return gameBinds, neededDevices, contexts
}

func populateImageOverlays(deviceMap common.DeviceMap, imageMap common.ImageMap,
	gameBinds fs2020BindsByDevice, data *common.GameData) common.OverlaysByImage {
	// Iterate through our game binds
	overlaysByImage := make(common.OverlaysByImage)
	for shortName, gameDevice := range gameBinds {
		inputs := deviceMap[shortName]
		image := imageMap[shortName]
		for context, actions := range gameDevice.ContextActions {
			for actionName, input := range actions {
				inputLookups := matchGameInputToModel(shortName, input, inputs,
					(*data).InputMap[shortName])

				for _, input := range inputLookups {
					inputData, found := inputs[input]
					if !found {
						log.Printf("Error: FS2020 unknown input to lookup %s for device %s\n",
							input, shortName)
					}
					if inputData.X == 0 && inputData.Y == 0 {
						log.Printf("Error: FS2020 location 0,0 for %s device %s %v\n",
							actionName, shortName, inputData)
						continue
					}
					common.GenerateImageOverlays(overlaysByImage, input, &inputData,
						gameData, actionName, context, shortName, image)
				}
			}
		}
	}

	return overlaysByImage
}

// matchGameInputToModel takes the game provided bindings with the device map to
// build a list of image overlays.
func matchGameInputToModel(deviceName string, actionData *fs2020Input,
	inputs common.DeviceInputs, gameInputMap common.InputTypeMapping) []string {
	inputLookups := make([]string, 0, 2)

	// First the primary input for this action
	input := matchGameInputToModelByRegex(deviceName,
		(*actionData).PrimaryInput, inputs, gameInputMap)
	if input != "" {
		inputLookups = append(inputLookups, input)
	} else {
		log.Printf("Error: FS2020 did not find primary input for %s\n", (*actionData).PrimaryInput)
	}
	// Now the secondary input
	if len((*actionData).SecondaryInput) > 0 {
		input := matchGameInputToModelByRegex(deviceName, (*actionData).SecondaryInput,
			inputs, gameInputMap)
		if input != "" {
			inputLookups = append(inputLookups, input)
		} else {
			log.Printf("Error: FS2020 did not find secondary input for %s\n",
				(*actionData).SecondaryInput)
		}
	}
	return inputLookups
}

// Matches an action to a device's inputs using regexes. Returns string to lookup input
func matchGameInputToModelByRegex(deviceName string, action string,
	inputs common.DeviceInputs, gameInputMap common.InputTypeMapping) string {
	var matches [][]string

	matches = regexes["Button"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		return matches[0][1]
	}

	matches = regexes["Axis"].FindAllStringSubmatch(action, -1)
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
	matches = regexes["Pov"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		direction := strings.Title(strings.ToLower(matches[0][2]))
		pov := fmt.Sprintf("POV%s%s", "1", direction)
		if len(matches[0][1]) > 0 {
			pov = fmt.Sprintf("POV%s%s", matches[0][1], direction)
		}
		return pov
	}

	matches = regexes["Rotation"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		rotation := fmt.Sprintf("R%sAxis", matches[0][1])
		if input, ok := gameInputMap["Rotation"]; ok {
			// Check override
			rotation = fmt.Sprintf("%sAxis", input[matches[0][1]])
		}
		return rotation
	}

	matches = regexes["Slider"].FindAllStringSubmatch(action, -1)
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

// FS2020 Input model
// Short name -> Context -> Action -> Primary/Secondary -> Key
type fs2020BindsByDevice map[string]*fs2020Device

type fs2020Device struct {
	DeviceFullName string
	ContextActions fs2020ContextActions
}

// Context -> Action
type fs2020ContextActions map[string]fs2020Actions

// Action -> Input
type fs2020Actions map[string]*fs2020Input

type fs2020Input struct {
	PrimaryInput   string
	SecondaryInput string
}

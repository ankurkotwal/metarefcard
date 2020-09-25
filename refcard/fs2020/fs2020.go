package fs2020

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ankurkotwal/MetaRef/refcard/data"
	"github.com/ankurkotwal/MetaRef/refcard/util"
)

var initiliased bool = false
var gameData *fs2020Data
var regexes map[string]*regexp.Regexp

// HandleRequest services the request to load files
func HandleRequest(files [][]byte, deviceMap data.DeviceMap, config *data.Config) (data.OverlaysByImage, map[string]string) {
	if !initiliased {
		gameData = loadGameModel(config.DebugOutput)
		regexes = make(map[string]*regexp.Regexp)
		for name, pattern := range gameData.Regexes {
			regexes[name] = regexp.MustCompile(pattern)
		}
		initiliased = true
	}
	gameBinds, contexts := loadInputFiles(files, gameData.DeviceNameMap, config.DebugOutput, config.VerboseOutput)

	neededDevices := make(map[string]bool)
	for device := range gameBinds {
		neededDevices[device] = true
	}
	deviceIndex := data.FilterDevices(deviceMap, neededDevices, config.DebugOutput)
	// Add device additions to the main device index
	for deviceName, deviceInputData := range gameData.InputOverrides {
		if deviceData, found := deviceIndex[deviceName]; found {
			for additionInput, additionData := range deviceInputData.Inputs {
				deviceData.Inputs[additionInput] = additionData
			}
		}
	}
	return populateImageOverlays(deviceIndex, gameBinds, gameData), contexts
}

// Load FS2020 specific data from our model. Update the device names (map game device name to our model names)
func loadGameModel(debugOutput bool) *fs2020Data {
	data := fs2020Data{}
	util.LoadYaml("configs/fs2020.yaml", &data, "FS2020 Data")

	fullToShort := deviceNameFullToShort{}
	// Update map of game device names to our model device names
	for fullName, shortName := range data.DeviceNameMap {
		if shortName != "" {
			fullToShort[fullName] = shortName
		} else {
			fullToShort[fullName] = deviceMissingInfo
		}
	}
	data.DeviceNameMap = fullToShort

	return &data
}

// Load the game config files (provided by user)
func loadInputFiles(files [][]byte, deviceNameMap deviceNameFullToShort,
	debugOutput bool, verboseOutput bool) (gameBindsByDevice, map[string]string) {
	gameBinds := make(gameBindsByDevice)
	contexts := make(map[string]string)

	// XML state variables
	var currentDevice *gameDevice
	var currentContext map[string]*gameAction
	var currentAction *gameAction
	currentKeyType := keyUnknown
	var currentKey *int

	for _, file := range files {
		decoder := xml.NewDecoder(bytes.NewReader(file))
		for {
			token, err := decoder.Token()
			if token == nil || err == io.EOF {
				// EOF means we're done.
				break
			} else if err != nil {
				log.Fatalf("Error decoding token: %s", err)
			}

			switch ty := token.(type) {
			case xml.StartElement:
				switch ty.Name.Local {
				case "Device":
					// Found new device
					var aDevice gameDevice
					for _, attr := range ty.Attr {
						switch attr.Name.Local {
						case "DeviceName":
							aDevice.DeviceName = attr.Value
							break
						case "GUID":
							aDevice.GUID = attr.Value
							break
						case "ProductID":
							aDevice.ProductID = attr.Value
							break
						}
					}
					var found bool
					currentDevice, found = gameBinds[aDevice.DeviceName]
					out, _ := json.Marshal(aDevice)
					if found {
						log.Printf("Duplicate device: %s\n", out)
					} else {
						if debugOutput {
							log.Printf("New device: %s\n", out)
						}
						currentDevice = &aDevice
						currentDevice.ContextActions = make(map[string]map[string]*gameAction)
						if shortName, ok := deviceNameMap[aDevice.DeviceName]; ok {
							gameBinds[shortName] = currentDevice
							if shortName == deviceMissingInfo {
								log.Printf("Error: Missing info for device '%s'\n", aDevice.DeviceName)
							}
						} else {
							deviceNameMap[shortName] = deviceUnknown
							log.Printf("Error: Unknown device '%s'\n", aDevice.DeviceName)
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
						log.Printf("Error: Duplicate context: %s\n", contextName)
					} else {
						if debugOutput {
							log.Printf("New context: %s\n", contextName)
						}
						currentContext = make(map[string]*gameAction)
						currentDevice.ContextActions[contextName] = currentContext
					}
				case "Action":
					// Found new action
					currentKeyType = keyUnknown
					var actionName string
					var action gameAction
					for _, attr := range ty.Attr {
						if attr.Name.Local == "ActionName" {
							actionName = attr.Value
						} else if attr.Name.Local == "Flag" {
							action.Flag, err = strconv.Atoi(attr.Value)
							if err != nil {
								log.Printf("Error: Action %s flag parsing error\n", actionName)
							}
						}
					}
					var found bool
					currentAction, found = currentContext[actionName]
					if found {
						log.Printf("Error: Duplicate action: %s\n", actionName)
					} else {
						if debugOutput {
							log.Printf("New action: %s\n", actionName)
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
								currentAction.PrimaryInfo = attr.Value
								currentKey = &currentAction.PrimaryKey
							case keySecondary:
								currentAction.SecondaryInfo = attr.Value
								currentKey = &currentAction.SecondaryKey
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
						log.Printf("Error: Primary key value %s parsing error\n", value)
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
			log.Printf("DeviceName=\"%s\" GUID=\"%s\" ProductId=\"%s\"\n",
				gameDevice.DeviceName, gameDevice.GUID, gameDevice.ProductID)
			for contextName, actions := range gameDevice.ContextActions {
				log.Printf("  ContextName=\"%s\"\n", contextName)
				for actionName, action := range actions {
					secondaryText := ""
					if len(action.SecondaryInfo) != 0 {
						secondaryText = fmt.Sprintf(" SecondaryInfo=\"%s\" SecondaryKey=\"%d\"",
							action.SecondaryInfo, action.SecondaryKey)
					}
					log.Printf("    ActionName=\"%s\" Flag=\"%d\" PrimaryInfo=\"%s\" PrimaryKey=\"%d\"%s\n",
						actionName, action.Flag, action.PrimaryInfo, action.PrimaryKey, secondaryText)
				}
			}
		}
	}

	return gameBinds, contexts
}

func populateImageOverlays(deviceIndex data.DeviceModel, gameBinds gameBindsByDevice,
	gameData *fs2020Data) data.OverlaysByImage {
	// Iterate through our game binds
	overlaysByImage := make(data.OverlaysByImage)
	for deviceName, gameDevice := range gameBinds {
		modelDevice := deviceIndex[deviceName]
		image := modelDevice.Image
		for context, actions := range gameDevice.ContextActions {
			for actionName, actionData := range actions {
				inputLookups := findMatchingInputModels(deviceName, actionData,
					modelDevice.Inputs, (*gameData).InputMap[deviceName])
				for _, input := range inputLookups {
					inputData, found := modelDevice.Inputs[input]
					if !found {
						log.Printf("Error: Unknown input to lookup %s for device %s\n", input, deviceName)
					}
					if inputData.ImageX == 0 && inputData.ImageY == 0 {
						log.Printf("Error: Location 0,0 for %s device %s %v ", actionName, deviceName, inputData)
						continue
					}
					var overlayData data.OverlayData
					overlayData.ContextToTexts = make(map[string][]string)
					overlayData.PosAndSize = &inputData
					var text string
					// Game data might have a better label for this text
					if label, found := (*gameData).InputLabels[actionName]; found {
						text = label
					} else {
						text = regexes["Key"].ReplaceAllString(actionName, "$1")
					}
					texts := make([]string, 1)
					texts[0] = text
					overlayData.ContextToTexts[context] = texts

					// Find by Image first
					deviceAndInput := fmt.Sprintf("%s:%s", deviceName, input)
					if overlay, found := overlaysByImage[image]; !found {
						// First time adding this image
						overlay := make(map[string]*data.OverlayData)
						overlaysByImage[image] = overlay
						overlay[deviceAndInput] = &overlayData
					} else {
						// Now find by input
						if previousOverlayData, found := overlay[deviceAndInput]; !found {
							// Not new image but new overlayData
							overlay[deviceAndInput] = &overlayData
						} else {
							// Concatenate input
							texts = append(previousOverlayData.ContextToTexts[context], text)
							sort.Strings(texts)
							previousOverlayData.ContextToTexts[context] = texts
						}
					}
				}
			}
		}
	}

	return overlaysByImage
}

// findMatchingInputModels takes the game provided bindings with the internal device map to
// build a list of image overlays.
func findMatchingInputModels(deviceName string, actionData *gameAction, inputs data.InputsMap,
	gameInputMap inputTypeMapping) []string {
	inputLookups := make([]string, 0)

	input := findMatchingInputModelsInner(deviceName,
		(*actionData).PrimaryInfo, inputs, gameInputMap)
	if input != "" {
		inputLookups = append(inputLookups, input)
	} else {
		log.Printf("Error: Did not find primary input for %s\n", (*actionData).PrimaryInfo)
	}
	if len((*actionData).SecondaryInfo) > 0 {
		input := findMatchingInputModelsInner(deviceName, (*actionData).SecondaryInfo, inputs, gameInputMap)
		if input != "" {
			inputLookups = append(inputLookups, input)
		} else {
			log.Printf("Error: Did not find secondary input for %s\n", (*actionData).SecondaryInfo)
		}
	}
	return inputLookups
}

// Matches an action to a device's inputs using regexes. Returns string to lookup input
func findMatchingInputModelsInner(deviceName string, action string, inputs data.InputsMap,
	gameInputMap inputTypeMapping) string {
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
			log.Printf("Error: Unknown action %s for slider on device %s\n", action, deviceName)
			return ""
		}
		if _, ok := inputs[slider]; ok {
			return slider
		}
		log.Printf("Error: Couldn't find slider %s on device %s\n", slider, deviceName)
		return ""

	}
	log.Printf("Error: Could not find matching Action %s on device %s\n", action, deviceName)
	return ""
}

type fs2020Data struct {
	DeviceNameMap  deviceNameFullToShort           `yaml:"DeviceNameMap"`
	InputMap       deviceInputTypeMapping          `yaml:"InputMapping"`
	InputOverrides map[string]data.DeviceInputData `yaml:"InputOverrides"`
	InputLabels    map[string]string               `yaml:"InputLabels"`
	Regexes        map[string]string               `yaml:"Regexes"`
}
type deviceNameFullToShort map[string]string
type deviceInputTypeMapping map[string]inputTypeMapping
type inputTypeMapping map[string]map[string]string // Device -> Type (Axis/Slider) -> axisInputMap/sliderInputMap
// Axis -> X/Y/Z -> model [R]X/Y/Z Axis
// Slider -> X/Y/Z -> model [R]U/V/X/Y/Z Axis

const (
	deviceUnknown     = "DeviceUnknown"     // Unfamiliar with this device
	deviceMissingInfo = "DeviceMissingInfo" // Only know the name of device
)

const (
	keyUnknown   = iota
	keyPrimary   = iota
	keySecondary = iota
)

// FS2020 Input model
// Device -> Context -> Action -> Primary/Secondary -> Key
type gameBindsByDevice map[string]*gameDevice

type gameDevice struct {
	DeviceName     string
	GUID           string
	ProductID      string
	ContextActions map[string]map[string]*gameAction
}

type gameAction struct {
	Flag          int
	PrimaryInfo   string
	PrimaryKey    int
	SecondaryInfo string
	SecondaryKey  int
}

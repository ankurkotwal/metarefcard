package fs2020

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ankurkotwal/InputRefCard/data"
	"github.com/ankurkotwal/InputRefCard/util"
	"golang.org/x/image/font"
)

var initiliased bool = false
var gameData *fs2020Data
var gameBinds gameBindsByDevice
var regexes map[string]*regexp.Regexp

// HandleRequest services the request to load files
func HandleRequest(generateImage func(data.OverlaysByImage, font.Face, data.Config),
	deviceMap data.DeviceMap, font font.Face, config data.Config, debugOutput bool, verboseOutput bool) {
	if !initiliased {
		// Load the game files provided
		gameData = loadGameModel(debugOutput)
		gameBinds = loadGameConfigs(flag.Args(), gameData.DeviceNameMap, debugOutput, verboseOutput)
		regexes = make(map[string]*regexp.Regexp)
		for name, pattern := range gameData.Regexes {
			regexes[name] = regexp.MustCompile(pattern)
		}
		initiliased = true
	}

	neededDevices := make(map[string]bool)
	for device := range gameBinds {
		neededDevices[device] = true
	}
	deviceIndex := data.FilterDevices(deviceMap, neededDevices, debugOutput)
	// Add device additions to the main device index
	for deviceName, deviceInputData := range gameData.InputAdditions {
		if deviceData, found := deviceIndex[deviceName]; found {
			for additionInput, additionData := range deviceInputData.Inputs {
				deviceData.Inputs[additionInput] = additionData
			}
		}
	}
	overlaysByImage := populateImageOverlays(deviceIndex, gameBinds, gameData)
	generateImage(overlaysByImage, font, config)
}

// Load FS2020 specific data from our model. Update the device names (map game device name to our model names)
func loadGameModel(debugOutput bool) *fs2020Data {
	data := fs2020Data{}
	util.LoadYaml("data/fs2020.yaml", &data,
		debugOutput, "FS2020 Data")

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
func loadGameConfigs(files []string, deviceNameMap deviceNameFullToShort,
	debugOutput bool, verboseOutput bool) gameBindsByDevice {
	gameBinds := make(gameBindsByDevice)

	// XML state variables
	var currentDevice *gameDevice
	var currentContext map[string]*gameAction
	var currentAction *gameAction
	currentKeyType := keyUnknown
	var currentKey *int

	for _, filename := range files {
		if debugOutput {
			log.Printf("Opening file %s\n", filename)
		}
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		decoder := xml.NewDecoder(file)
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
	return gameBinds
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
						log.Printf("Error: Unknown input to lookup %s\n", input)
					}
					if inputData.ImageX == 0 && inputData.ImageY == 0 {
						log.Printf("Error: Location 0,0 for %s %v", actionName, inputData)
						continue
					}
					var overlayData data.OverlayData
					overlayData.Context = context
					// Game data might have a better label for this text
					if label, found := (*gameData).InputLabels[actionName]; found {
						overlayData.Text = label
					} else {
						overlayData.Text = actionName
					}
					overlayData.PosAndSize = &inputData

					// Find by Image first
					if overlay, found := overlaysByImage[image]; !found {
						overlay := make(map[string]data.OverlayData)
						overlay[input] = overlayData
						overlaysByImage[image] = overlay
					} else {
						// Now find by input
						if previousOverlayData, found := overlay[input]; !found {
							overlay[input] = overlayData
						} else {
							// Concatenate input
							overlayData.Text = fmt.Sprintf("%s   %s",
								previousOverlayData.Text, overlayData.Text)
							overlay[input] = overlayData
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
		return rotation
	}

	matches = regexes["Slider"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		var slider string
		// TODO - Slider map.
		if input, ok := gameInputMap[deviceName]; ok {
			_ = input
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
	InputAdditions map[string]data.DeviceInputData `yaml:"InputAdditions"`
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

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

	"github.com/ankurkotwal/MetaRefCard/metarefcard/common"
)

var initiliased bool = false
var gameData *common.GameData
var regexes common.RegexByName

// HandleRequest services the request to load files
func HandleRequest(files [][]byte, deviceMap common.DeviceMap,
	deviceNameToImage common.DeviceNameToImage,
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
	deviceIndex := common.OrgDeviceModel(deviceMap, neededDevices, config)
	common.GenerateContextColours(contexts, config)
	return populateImageOverlays(deviceIndex, gameBinds, gameData), contexts
}

// Load the game config files (provided by user)
func loadInputFiles(files [][]byte, deviceNameMap common.DeviceNameFullToShort,
	debugOutput bool, verboseOutput bool) (fs2020BindsByDevice,
	common.MockSet, common.MockSet) {
	gameBinds := make(fs2020BindsByDevice)
	devices := make(common.MockSet)
	contexts := make(common.MockSet)

	// XML state variables
	var currentDevice *fs2020Device
	var currentContext map[string]*fs2020Action
	var currentAction *fs2020Action
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
						log.Printf("Error: FS2020 duplicate device: %s\n", out)
					} else {
						if debugOutput {
							log.Printf("Error: FS2020 new device: %s\n", out)
						}
						currentDevice = &aDevice
						currentDevice.ContextActions = make(map[string]map[string]*fs2020Action)
						if shortName, ok := deviceNameMap[aDevice.DeviceName]; ok {
							devices[shortName] = ""
							gameBinds[shortName] = currentDevice
							if shortName == common.DeviceMissingInfo {
								log.Printf("Error: FS2020 missing info for device '%s'\n",
									aDevice.DeviceName)
							}
						} else {
							deviceNameMap[shortName] = common.DeviceUnknown
							log.Printf("Error: FS2020 unknown device '%s'\n", aDevice.DeviceName)
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
						currentContext = make(map[string]*fs2020Action)
						currentDevice.ContextActions[contextName] = currentContext
					}
				case "Action":
					// Found new action
					currentKeyType = keyUnknown
					var actionName string
					var action fs2020Action
					for _, attr := range ty.Attr {
						if attr.Name.Local == "ActionName" {
							actionName = attr.Value
						} else if attr.Name.Local == "Flag" {
							action.Flag, err = strconv.Atoi(attr.Value)
							if err != nil {
								log.Printf("Error: FS2020 action %s flag parsing error\n", actionName)
							}
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

	return gameBinds, devices, contexts
}

func populateImageOverlays(deviceIndex common.DeviceModel, gameBinds fs2020BindsByDevice,
	data *common.GameData) common.OverlaysByImage {
	// Iterate through our game binds
	overlaysByImage := make(common.OverlaysByImage)
	for deviceName, gameDevice := range gameBinds {
		modelDevice := deviceIndex[deviceName]
		image := modelDevice.Image
		for context, actions := range gameDevice.ContextActions {
			for actionName, actionData := range actions {
				inputLookups := findMatchingInputModels(deviceName, actionData,
					modelDevice.Inputs, (*data).InputMap[deviceName])
				for _, input := range inputLookups {
					inputData, found := modelDevice.Inputs[input]
					if !found {
						log.Printf("Error: FS2020 unknown input to lookup %s for device %s\n",
							input, deviceName)
					}
					if inputData.ImageX == 0 && inputData.ImageY == 0 {
						log.Printf("Error: FS2020 location 0,0 for %s device %s %v\n",
							actionName, deviceName, inputData)
						continue
					}
					var overlayData common.OverlayData
					overlayData.ContextToTexts = make(map[string][]string)
					overlayData.PosAndSize = &inputData
					var text string
					// Game data might have a better label for this text
					if label, found := (*data).InputLabels[actionName]; found {
						text = label
					} else {
						text = actionName
						log.Printf("Error: FS2020 Unknown action %s context %s device %s\n",
							actionName, context, deviceName)
					}
					texts := make([]string, 1)
					texts[0] = text
					overlayData.ContextToTexts[context] = texts

					// Find by Image first
					deviceAndInput := fmt.Sprintf("%s:%s", deviceName, input)
					if overlay, found := overlaysByImage[image]; !found {
						// First time adding this image
						overlay := make(map[string]*common.OverlayData)
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
func findMatchingInputModels(deviceName string, actionData *fs2020Action,
	inputs common.InputsMap, gameInputMap common.InputTypeMapping) []string {
	inputLookups := make([]string, 0)

	input := findMatchingInputModelsByRegex(deviceName,
		(*actionData).PrimaryInfo, inputs, gameInputMap)
	if input != "" {
		inputLookups = append(inputLookups, input)
	} else {
		log.Printf("Error: FS2020 did not find primary input for %s\n", (*actionData).PrimaryInfo)
	}
	if len((*actionData).SecondaryInfo) > 0 {
		input := findMatchingInputModelsByRegex(deviceName, (*actionData).SecondaryInfo,
			inputs, gameInputMap)
		if input != "" {
			inputLookups = append(inputLookups, input)
		} else {
			log.Printf("Error: FS2020 did not find secondary input for %s\n",
				(*actionData).SecondaryInfo)
		}
	}
	return inputLookups
}

// Matches an action to a device's inputs using regexes. Returns string to lookup input
func findMatchingInputModelsByRegex(deviceName string, action string,
	inputs common.InputsMap, gameInputMap common.InputTypeMapping) string {
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
// Device -> Context -> Action -> Primary/Secondary -> Key
type fs2020BindsByDevice map[string]*fs2020Device

type fs2020Device struct {
	DeviceName     string
	GUID           string
	ProductID      string
	ContextActions map[string]map[string]*fs2020Action
}

type fs2020Action struct {
	Flag          int
	PrimaryInfo   string
	PrimaryKey    int
	SecondaryInfo string
	SecondaryKey  int
}

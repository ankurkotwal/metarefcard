package sws

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ankurkotwal/MetaRefCard/metarefcard/common"
)

var initiliased bool = false
var regexes swsRegexes
var gameData *common.GameData

// HandleRequest services the request to load files
func HandleRequest(files [][]byte, deviceMap common.DeviceMap,
	config *common.Config) (common.OverlaysByImage, map[string]string) {
	if !initiliased {
		gameData = common.LoadGameModel("config/sws.yaml",
			"StarWarsSquadrons Data", config.DebugOutput)
		for name, pattern := range gameData.Regexes {
			switch name {
			case "BindStarship":
				regexes.BindStarship = regexp.MustCompile(pattern)
			case "BindSoldier":
				regexes.BindSoldier = regexp.MustCompile(pattern)
			case "BindDefault":
				regexes.BindDefault = regexp.MustCompile(pattern)
			case "Joystick":
				regexes.Joystick = regexp.MustCompile(pattern)
			default:
				log.Printf("Error: SWS Unknown Regex %s pattern %s\n", name, pattern)
			}
		}
		initiliased = true
	}

	gameBinds, contexts := loadInputFiles(files, gameData.DeviceNameMap,
		config.DebugOutput, config.VerboseOutput)

	neededDevices := make(map[string]bool)
	for device := range gameBinds {
		neededDevices[device] = true
	}
	deviceIndex := common.FilterDevices(deviceMap, neededDevices, config.DebugOutput)
	// Add device additions to the main device index
	for deviceName, deviceInputData := range config.InputOverrides {
		if deviceData, found := deviceIndex[deviceName]; found {
			for additionInput, additionData := range deviceInputData.Inputs {
				deviceData.Inputs[additionInput] = additionData
			}
		}
	}

	// Generate colours for contexts here
	categories := common.GenerateContextColours(contexts, config)

	return populateImageOverlays(deviceIndex, gameBinds, gameData), categories
}

// Load the game config files (provided by user)
func loadInputFiles(files [][]byte, deviceNameMap common.DeviceNameFullToShort,
	debugOutput bool, verboseOutput bool) (swsBindsByDevice, []string) {
	gameBinds := make(swsBindsByDevice)
	contexts := make(map[string]bool)

	// deviceIndex: deviceId -> full name
	deviceIndex := make(map[string]string)
	contextActionIndex := make(swsContextActionIndex)

	// Load all the device and inputs
	var matches [][]string
	for idx, file := range files {
		scanner := bufio.NewScanner(bytes.NewReader(file))
		for scanner.Scan() {
			line := scanner.Text()

			matches = regexes.BindStarship.FindAllStringSubmatch(line, -1)
			if matches != nil {
				addAction(contextActionIndex, "Starship", contexts, matches[0][1],
					matches[0][2], matches[0][3], matches[0][4])
			}
			matches = regexes.BindSoldier.FindAllStringSubmatch(line, -1)
			if matches != nil {
				addAction(contextActionIndex, "Soldier", contexts, matches[0][1],
					matches[0][2], matches[0][3], matches[0][4])
			}
			matches = regexes.BindDefault.FindAllStringSubmatch(line, -1)
			if matches != nil {
				addAction(contextActionIndex, "Default", contexts, matches[0][1],
					matches[0][2], matches[0][3], matches[0][4])
			}
			matches = regexes.Joystick.FindAllStringSubmatch(line, -1)
			if matches != nil && len(matches[0][2]) > 0 {
				if shortName, found := deviceNameMap[matches[0][2]]; !found {
					log.Printf("Error: SWS Unknown device found %s\n", matches[0][2])
					continue
				} else {
					num, err := strconv.Atoi(matches[0][1])
					// Subtract 1 from the Joystick index to match deviceIds in the file
					num--
					if err == nil && num >= 0 {
						deviceIndex[strconv.Itoa(num)] = shortName
					} else {
						log.Printf("Error: SWS unexpected device number %s\n", matches[0][1])
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error: SWS scan file %d. %s\n", idx, err)
		}
	}

	// Now iterate through the object to build our internal index.
	// We do it in multiple passes to avoid having to make assumptions around
	// the order of fields in the game's config files.
	for context, actionMap := range contextActionIndex {
		for action, actionSubMap := range actionMap {
			// Get the device first
			deviceID, found := actionSubMap["deviceid"]
			if !found {
				log.Printf("Error: SWS couldn't find deviceId in %s->%s->%v\n", context, action, actionSubMap)
				continue
			}
			shortName, found := deviceIndex[deviceID]
			if !found {
				continue // We only care about devices in deviceIndex
			}
			// Add this button to index
			contextActions, found := gameBinds[shortName]
			if !found {
				contextActions = make(swsContextActions)
				gameBinds[shortName] = contextActions
			}
			actions, found := contextActions[context]
			if !found {
				actions = make(swsActions)
				contextActions[context] = actions
			}

			// Build the action details
			actionDetails := swsActionDetails{}
			for actionSub, value := range actionSubMap {
				field := getInputTypeAsField(actionSub, &actionDetails)
				if field == nil {
					log.Printf("Error: SWS unknown inputType %s value %s\n",
						actionSub, value)
				} else {
					*field = value
				}
			}

			// Assign action details accordingly
			input := interpretInput(&actionDetails)
			if len(input) > 0 {
				actions[action] = input
			} else {
				delete(actions, action)
			}
		}
	}

	contextsArray := make([]string, 0, len(contexts))
	for context := range contexts {
		contextsArray = append(contextsArray, context)
	}

	return gameBinds, contextsArray
}

func addAction(contextActionIndex swsContextActionIndex,
	context string, contexts map[string]bool, action string, deviceNum string,
	actionSub string, value string) {
	contexts[context] = true
	if context == "" {
		log.Printf("blah\n")
	}
	var found bool
	var actionMap map[string]map[string]string
	if actionMap, found = contextActionIndex[context]; !found {
		// First time for this context
		actionMap = make(map[string]map[string]string)
		contextActionIndex[context] = actionMap
	}
	var actionSubMap map[string]string
	if actionSubMap, found = actionMap[action]; !found {
		// First time for this device number
		actionSubMap = make(map[string]string)
		actionMap[action] = actionSubMap
	}
	actionSubMap[actionSub] = value
}

func getInputTypeAsField(actionSub string, currAction *swsActionDetails) *string {
	actionSub = strings.ToLower(actionSub)
	switch actionSub {
	case "altbutton":
		return &currAction.AltButton
	case "axis":
		return &currAction.Axis
	case "button":
		return &currAction.Button
	case "deviceid":
		return &currAction.DeviceID
	case "identifier":
		return &currAction.Identifier
	case "modifier":
		return &currAction.Modifier
	case "negate":
		return &currAction.Negate
	case "type":
		return &currAction.Type
	}
	return nil
}

func interpretInput(details *swsActionDetails) string {
	switch details.Axis {
	case "8":
		return "XAxis" // Throttle
	case "9":
		return "YAxis" // Stick
	case "10":
		return "XAxis" // Stick
	case "26":
		switch details.Button {
		case "46":
			fallthrough
		case "47":
			return "RZAxis" // Stick
		case "48":
			return "POV1Up" // Stick
		case "49":
			return "POV1Down" // Stick
		case "50":
			return "POV1Left" // Stick
		case "51":
			return "POV1Right" // Stick
		case "73":
			return "28" // Throttle Pinky Rocker Up
		case "74":
			return "29" // Throttle Pinky Rocker Down
		case "80":
			fallthrough // Stick TODO
		case "86":
			return ""
		}
		button, err := strconv.Atoi(details.Button)
		if err == nil {
			button -= 21 // Seems like a hardcoded number?
		}
		return strconv.Itoa(button)
	}
	log.Printf("Error SWS unknown input %v\n", details)
	return ""
}

func populateImageOverlays(deviceIndex common.DeviceModel, gameBinds swsBindsByDevice,
	data *common.GameData) common.OverlaysByImage {
	// Iterate through our game binds
	overlaysByImage := make(common.OverlaysByImage)
	for deviceName, contextActions := range gameBinds {
		modelDevice := deviceIndex[deviceName]
		image := modelDevice.Image
		for context, actions := range contextActions {
			for action, input := range actions {
				inputData, found := modelDevice.Inputs[input]
				if !found {
					log.Printf("Error: SWS unknown input to lookup %s for device %s\n",
						input, deviceName)
				}
				if inputData.ImageX == 0 && inputData.ImageY == 0 {
					log.Printf("Error: Location 0,0 for %s device %s %v ",
						action, deviceName, inputData)
					continue
				}
				var overlayData common.OverlayData
				overlayData.ContextToTexts = make(map[string][]string)
				overlayData.PosAndSize = &inputData
				var text string
				// Game data might have a better label for this text
				if label, found := (*data).InputLabels[action]; found {
					text = label
				} else {
					text = action
					log.Printf("Unknown action %s context %s device %s",
						action, context, deviceName)
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

	return overlaysByImage
}

// swsContextActionIndex: context -> action name -> action sub -> value
type swsContextActionIndex map[string]map[string]map[string]string

type swsRegexes struct {
	BindStarship *regexp.Regexp
	BindSoldier  *regexp.Regexp
	BindDefault  *regexp.Regexp
	Joystick     *regexp.Regexp
}

// Device short name -> ContextAction
type swsBindsByDevice map[string]swsContextActions

// Context -> Actions
type swsContextActions map[string]swsActions

// Action -> Input
type swsActions map[string]string

// Parsed fields from sws config
type swsActionDetails struct {
	AltButton  string
	Axis       string
	Button     string
	DeviceID   string
	Identifier string
	Modifier   string
	Negate     string
	Type       string
}

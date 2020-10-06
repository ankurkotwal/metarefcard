package sws

import (
	"bufio"
	"bytes"
	"log"
	"regexp"
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
	for deviceName, deviceInputData := range gameData.InputOverrides {
		if deviceData, found := deviceIndex[deviceName]; found {
			for additionInput, additionData := range deviceInputData.Inputs {
				deviceData.Inputs[additionInput] = additionData
			}
		}
	}

	// Generate colours for contexts here
	categories := common.GenerateContextColours(contexts, config)

	// TODO
	// return populateImageOverlays(deviceIndex, gameBinds, gameData), categories
	return nil, categories
}

// Load the game config files (provided by user)
func loadInputFiles(files [][]byte, deviceNameMap common.DeviceNameFullToShort,
	debugOutput bool, verboseOutput bool) (swsBindsByDevice, []string) {
	gameBinds := make(swsBindsByDevice)
	contexts := make(map[string]bool)

	// deviceIndex: deviceId -> full name
	deviceIndex := make(map[string]string)
	// contextActionDeviceMap: context -> action name -> device id -> inputType -> value
	contextActionDeviceMap := make(map[string]map[string]map[string]map[string]string)

	// Load all the device and inputs
	var matches [][]string
	for idx, file := range files {
		scanner := bufio.NewScanner(bytes.NewReader(file))
		for scanner.Scan() {
			line := scanner.Text()

			matches = regexes.BindStarship.FindAllStringSubmatch(line, -1)
			if matches != nil {
				addAction(contextActionDeviceMap, "Starship", contexts, matches[0][1],
					matches[0][2], matches[0][3], matches[0][4])
			}
			matches = regexes.BindSoldier.FindAllStringSubmatch(line, -1)
			if matches != nil {
				addAction(contextActionDeviceMap, "Soldier", contexts, matches[0][1],
					matches[0][2], matches[0][3], matches[0][4])
			}
			matches = regexes.BindDefault.FindAllStringSubmatch(line, -1)
			if matches != nil {
				addAction(contextActionDeviceMap, "Default", contexts, matches[0][1],
					matches[0][2], matches[0][3], matches[0][4])
			}
			matches = regexes.Joystick.FindAllStringSubmatch(line, -1)
			if matches != nil {
				if len(matches[0][2]) > 0 {
					if shortName, found := deviceNameMap[matches[0][2]]; !found {
						log.Printf("Error: SWS Unknown device found %s\n", matches[0][2])
					} else {
						deviceIndex[matches[0][1]] = shortName
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error: SWS scan file %d. %s\n", idx, err)
		}
	}

	var contextAction swsContextAction
	var actions swsActions
	var actionDetails *swsActionDetails
	for context, actionDeviceMap := range contextActionDeviceMap {
		for action, deviceMap := range actionDeviceMap {
			for deviceNum, deviceActionDetails := range deviceMap {
				var found bool
				var shortName string
				if shortName, found = deviceIndex[deviceNum]; !found {
					continue
				}
				if contextAction, found = gameBinds[shortName]; !found {
					contextAction = make(swsContextAction)
					gameBinds[shortName] = contextAction
				}
				if actions, found = contextAction[context]; !found {
					actions = make(swsActions)
					contextAction[context] = actions
				}
				if actionDetails, found = actions[action]; !found {
					actionDetails = &swsActionDetails{}
					actions[action] = actionDetails
				}
				for inputType, value := range deviceActionDetails {
					field := getInputTypeAsField(inputType, actionDetails)
					if field == nil {
						log.Printf("Error: SWS unknown inputType %s value %s", inputType, value)
					} else {
						*field = value
					}
				}
			}
		}
	}

	contextsArray := make([]string, len(contexts))
	for context := range contexts {
		contextsArray = append(contextsArray, context)
	}

	common.PrintYamlObject(gameBinds, "SWS gamebinds")
	return gameBinds, contextsArray
}

func addAction(contextActionDeviceMap map[string]map[string]map[string]map[string]string,
	context string, contexts map[string]bool, action string, deviceNum string,
	inputType string, value string) {
	contexts[context] = true
	var found bool
	var actionDeviceMap map[string]map[string]map[string]string
	if actionDeviceMap, found = contextActionDeviceMap[context]; !found {
		// First time for this context
		actionDeviceMap = make(map[string]map[string]map[string]string)
		contextActionDeviceMap[context] = actionDeviceMap
	}
	var deviceMap map[string]map[string]string
	if deviceMap, found = actionDeviceMap[action]; !found {
		// First time for this action
		deviceMap = make(map[string]map[string]string)
		actionDeviceMap[action] = deviceMap
	}
	var deviceActionDetails map[string]string
	if deviceActionDetails, found = deviceMap[deviceNum]; !found {
		// First time for this device number
		deviceActionDetails = make(map[string]string)
		deviceMap[deviceNum] = deviceActionDetails
	}
	deviceActionDetails[inputType] = value
}

func getInputTypeAsField(inputType string, currAction *swsActionDetails) *string {
	inputType = strings.ToLower(inputType)
	switch inputType {
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

type swsRegexes struct {
	BindStarship *regexp.Regexp
	BindSoldier  *regexp.Regexp
	BindDefault  *regexp.Regexp
	Joystick     *regexp.Regexp
}

// Device short name -> ContextAction
type swsBindsByDevice map[string]swsContextAction

// Context -> Actions
type swsContextAction map[string]swsActions

// Action -> Details
type swsActions map[string]*swsActionDetails

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

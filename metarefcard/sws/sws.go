package sws

import (
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ankurkotwal/MetaRefCard/metarefcard/common"
)

var firstInit sync.Once
var sharedRegexes swsRegexes
var sharedGameData common.GameData

const (
	label = "sws"
	desc  = "Star Wars Squadrons input configs"
)

// GetGameInfo returns the info needed to fit into MetaRefCard
// Returns:
//   * Game label / name
//   * User friendly command line description
//   * Func handler for incoming request
//   * Func that matches the game input format to MRC's model
func GetGameInfo() (string, string, common.FuncRequestHandler, common.FuncMatchGameInputToModel) {
	return label, desc, handleRequest, matchGameInputToModel
}

// handleRequest services the request to load files
func handleRequest(files [][]byte, config *common.Config, log *common.Logger) (common.GameData,
	common.GameBindsByProfile, common.MockSet, common.MockSet, string) {
	firstInit.Do(func() {
		sharedGameData = common.LoadGameModel("config/sws.yaml",
			"StarWarsSquadrons Data", config.DebugOutput, log)
		sharedRegexes.Bind = regexp.MustCompile(sharedGameData.Regexes["Bind"])
		sharedRegexes.Joystick = regexp.MustCompile(sharedGameData.Regexes["Joystick"])
	})

	gameBinds, gameDevices, gameContexts := loadInputFiles(files, config.Devices.DeviceToShortNameMap,
		log, config.DebugOutput, config.VerboseOutput)
	common.GenerateContextColours(gameContexts, config)
	return sharedGameData, gameBinds, gameDevices, gameContexts, sharedGameData.Logo
}

// Load the game config files (provided by user)
func loadInputFiles(files [][]byte, deviceNameMap common.DeviceNameFullToShort,
	log *common.Logger, bool, verboseOutput bool) (common.GameBindsByProfile,
	common.MockSet, common.MockSet) {
	gameBindsByProfile := make(common.GameBindsByProfile)
	gameBinds := make(common.GameDeviceContextActions)
	gameBindsByProfile[common.ProfileDefault] = gameBinds
	deviceNames := make(common.MockSet)
	contexts := make(common.MockSet)

	// deviceIndex: deviceId -> full name
	deviceIndex := make(map[string]string)
	contextActionIndex := make(swsContextActionIndex)

	// Load all the device and inputs
	var matches [][]string
	for idx, file := range files {
		scanner := bufio.NewScanner(bytes.NewReader(file))
		for scanner.Scan() {
			line := scanner.Text()

			matches = sharedRegexes.Bind.FindAllStringSubmatch(line, -1)
			if matches != nil {
				addAction(contextActionIndex, matches[0][1], contexts, matches[0][2],
					matches[0][3], matches[0][4], matches[0][5])
			}
			matches = sharedRegexes.Joystick.FindAllStringSubmatch(line, -1)
			if matches != nil && len(matches[0][2]) > 0 {
				if shortName, found := deviceNameMap[matches[0][2]]; !found {
					log.Err("SWS Unknown device found %s", matches[0][2])
					continue
				} else {
					num, err := strconv.Atoi(matches[0][1])
					// Subtract 1 from the Joystick index to match deviceIds in the file
					num--
					if err == nil && num >= 0 {
						deviceIndex[strconv.Itoa(num)] = shortName
						deviceNames[shortName] = ""
					} else {
						log.Err("SWS unexpected device number %s", matches[0][1])
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Err("SWS scan file %d. %s", idx, err)
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
				log.Err("SWS couldn't find deviceId in %s->%s->%v", context, action, actionSubMap)
				continue
			}
			shortName, found := deviceIndex[deviceID]
			if !found {
				continue // We only care about devices in deviceIndex
			}
			// Add this button to index
			contextActions, found := gameBinds[shortName]
			if !found {
				contextActions = make(common.GameContextActions)
				gameBinds[shortName] = contextActions
			}
			actions, found := contextActions[context]
			if !found {
				actions = make(common.GameActions)
				contextActions[context] = actions
			}

			// Build the action details
			actionDetails := swsActionDetails{}
			for actionSub, value := range actionSubMap {
				field := getInputTypeAsField(actionSub, &actionDetails)
				if field == nil {
					log.Err("SWS unknown inputType %s value %s",
						actionSub, value)
				} else {
					*field = value
				}
			}

			// Assign action details accordingly
			input := interpretInput(&actionDetails, shortName, context, action, log)
			if len(input) > 0 {
				gameAction, found := actions[action]
				if !found {
					gameAction = make(common.GameInput, common.NumInputs)
					actions[action] = gameAction
				}
				gameAction[common.InputPrimary] = input
			} else {
				delete(actions, action)
			}
		}
	}

	return gameBindsByProfile, deviceNames, contexts
}

func addAction(contextActionIndex swsContextActionIndex,
	context string, contexts common.MockSet, action string, deviceNum string,
	actionSub string, value string) {
	contexts[context] = ""

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

func interpretInput(details *swsActionDetails, device string, context string,
	action string, log *common.Logger) string {
	// TODO - Currently hardcoded for the X-55 based on reverse engineering.
	switch details.Axis {
	case "8":
		return "XAxis" // Throttle
	case "9":
		return "YAxis" // Stick
	case "10":
		return "XAxis" // Stick
	case "11":
		return "YAxis" // Stick
	case "26":
		button, err := strconv.Atoi(details.Button)
		if err == nil {
			if button > 21 && button < 40 {
				button -= 21 // Seems like a hardcoded number?
				return strconv.Itoa(button)
			} else if button >= 64 && button < 86 {
				button -= 45 // Another hardcoded number
				return strconv.Itoa(button)
			} else if button == 86 {
				return ""
			}
			switch device {
			case "SaitekX55Joystick":
				switch button {
				case 46:
					return "RZAxis"
				case 47:
					return "RZAxis"
				case 48:
					return "POV1Up"
				case 49:
					return "POV1Down"
				case 50:
					return "POV1Left"
				case 51:
					return "POV1Right"
				}
			case "SaitekX55Throttle":
				switch button {
				case 40:
					return "ZAxis"
				case 41:
					return "ZAxis"
				case 42:
					return "RXAxis"
				case 43:
					return "RXAxis"
				case 44:
					return "RYAxis"
				case 45:
					return "RYAxis"
				case 46:
					return "RZAxis"
				case 47:
					return "RZAxis"
				}
			}
		}
	}
	log.Err("SWS Unknown input - device %s context %s action %s data %v",
		device, context, action, details)
	return ""
}

// matchGameInputToModel - returns a common.GameInput of the inputs that can be displayed.
// Also returns the label to use for error text
func matchGameInputToModel(deviceName string, gameInput common.GameInput,
	deviceInputs common.DeviceInputs, gameInputMap common.InputTypeMapping,
	log *common.Logger) (common.GameInput, string) {
	// For SWS, we've already got the structure right
	return gameInput, sharedGameData.Logo
}

// swsContextActionIndex: context -> action name -> action sub -> value
type swsContextActionIndex map[string]map[string]map[string]string

type swsRegexes struct {
	Bind     *regexp.Regexp
	Joystick *regexp.Regexp
}

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

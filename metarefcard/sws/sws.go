package sws

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	mrc "github.com/ankurkotwal/MetaRefCard/metarefcard/common"
)

var firstInit sync.Once
var sharedRegexes swsRegexes
var sharedGameData mrc.GameData

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
func GetGameInfo() (string, string, mrc.FuncRequestHandler, mrc.FuncMatchGameInputToModel) {
	return label, desc, handleRequest, matchGameInputToModel
}

// handleRequest services the request to load files
func handleRequest(files [][]byte, cfg *mrc.Config, log *mrc.Logger) (mrc.GameData,
	mrc.GameBindsByProfile, mrc.Set, mrc.ContextToColours, string) {
	firstInit.Do(func() {
		sharedGameData = mrc.LoadGameModel("config/sws.yaml", "StarWarsSquadrons Data",
			cfg.DebugOutput, log)
		sharedRegexes.Bind = regexp.MustCompile(sharedGameData.Regexes["Bind"])
		sharedRegexes.Joystick = regexp.MustCompile(sharedGameData.Regexes["Joystick"])
	})

	gameBinds, gameDevices, gameContexts := loadInputFiles(files, cfg.Devices.DeviceToShortNameMap,
		log, cfg.DebugOutput, cfg.VerboseOutput)
	mrc.GenerateContextColours(gameContexts, cfg)
	return sharedGameData, gameBinds, gameDevices, gameContexts, sharedGameData.Logo
}

// Load the game config files (provided by user)
func loadInputFiles(files [][]byte, deviceNameMap mrc.DeviceNameFullToShort, log *mrc.Logger, bool,
	verboseOutput bool) (mrc.GameBindsByProfile, mrc.Set, mrc.ContextToColours) {
	gameBindsByProfile := make(mrc.GameBindsByProfile)
	gameBinds := make(mrc.GameDeviceContextActions)
	gameBindsByProfile[mrc.ProfileDefault] = gameBinds
	deviceNames := make(mrc.Set)
	contexts := make(mrc.ContextToColours)

	// deviceIndex: deviceId -> full name
	deviceIndex := make(map[string]string)
	contextActionIndex := make(swsContextActionIndex)

	// Load all the device and inputs
	for idx, file := range files {
		scanner := bufio.NewScanner(bytes.NewReader(file))
		for scanner.Scan() {
			line := scanner.Text()

			matches := sharedRegexes.Bind.FindAllStringSubmatch(line, -1)
			if matches != nil {
				addAction(contextActionIndex, matches[0][1], contexts, matches[0][2], matches[0][3],
					matches[0][4], matches[0][5])
				continue
			}
			matches2 := sharedRegexes.Joystick.FindAllStringSubmatch(line, -1)
			if matches2 != nil && len(matches2[0][2]) > 0 {
				if shortName, found := deviceNameMap[matches2[0][2]]; !found {
					log.Err("SWS Unknown device found %s", matches2[0][2])
					continue
				} else {
					num, err := strconv.Atoi(matches2[0][1])
					// Subtract 1 from the Joystick index to match deviceIds in the file
					num--
					if err == nil && num >= 0 {
						deviceIndex[strconv.Itoa(num)] = shortName
						deviceNames[shortName] = true
					} else {
						log.Err("SWS unexpected device number %s", matches2[0][1])
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
				contextActions = make(mrc.GameContextActions)
				gameBinds[shortName] = contextActions
			}
			actions, found := contextActions[context]
			if !found {
				actions = make(mrc.GameActions)
				contextActions[context] = actions
			}

			// Build the action details
			actionDetails := swsActionDetails{}
			for actionSub, value := range actionSubMap {
				field, err := getInputTypeAsField(actionSub, &actionDetails)
				if err != nil {
					log.Err(fmt.Sprintf("%s value %s", err, value))
				} else if field != nil { // Ignore nil fields, this is unneded information
					*field = value
				}
			}

			// Assign action details accordingly
			input := interpretInput(&actionDetails, shortName, context, action, log)
			if len(input) > 0 {
				gameAction, found := actions[action]
				if !found {
					gameAction = make(mrc.GameInput, mrc.NumInputs)
					actions[action] = gameAction
				}
				gameAction[mrc.InputPrimary] = input
			} else {
				delete(actions, action)
			}
		}
	}

	return gameBindsByProfile, deviceNames, contexts
}

func addAction(contextActionIndex swsContextActionIndex, context string,
	contexts mrc.ContextToColours, action string, deviceNum string,
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

func getInputTypeAsField(actionSub string, currAction *swsActionDetails) (*string, error) {
	actionSub = strings.ToLower(actionSub)
	switch actionSub {
	case "axis":
		return &currAction.Axis, nil
	case "button":
		return &currAction.Button, nil
	case "altbutton", "deviceid", "identifier", "modifier", "negate", "type":
		// Don't need to store these but they aren't an error
		return nil, nil
	}
	// Unknown actionSub found
	return nil, fmt.Errorf("SWS unknown inputType %s", actionSub)
}

func interpretInput(details *swsActionDetails, device string, context string, action string,
	log *mrc.Logger) string {
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

// matchGameInputToModel - returns a mrc.GameInput of the inputs that can be displayed.
// Also returns the label to use for error text
func matchGameInputToModel(deviceName string, gameInput mrc.GameInput,
	deviceInputs mrc.DeviceInputs, gameInputMap mrc.InputTypeMapping,
	log *mrc.Logger) (mrc.GameInput, string) {
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
	// Unused field AltButton  string
	Axis   string
	Button string
	// Unused field DeviceID   string
	// Unused field Identifier string
	// Unused field Modifier   string
	// Unused field Negate     string
	// Unused field Type       string
}

package sws

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/ankurkotwal/metarefcard/mrc/common"
)

var firstInit sync.Once
var sharedRegexes swsRegexes
var sharedGameData common.GameData

// ScannerReader interface for bufio.Scanner - allows injection for testing
type ScannerReader interface {
	Scan() bool
	Text() string
	Err() error
}

// scannerFactory creates a ScannerReader from bytes (can be replaced in tests)
var scannerFactory = func(data []byte) ScannerReader {
	return bufio.NewScanner(bytes.NewReader(data))
}

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
func handleRequest(files [][]byte, cfg *common.Config, log *common.Logger) (common.GameData,
	common.GameBindsByProfile, common.Set, common.ContextToColours, string) {
	firstInit.Do(func() {
		sharedGameData = common.LoadGameModel("config/sws.yaml", "StarWarsSquadrons Data",
			cfg.DebugOutput, log)
		sharedRegexes.Bind = regexp.MustCompile(sharedGameData.Regexes["Bind"])
		sharedRegexes.Joystick = regexp.MustCompile(sharedGameData.Regexes["Joystick"])
	})

	gameBinds, gameDevices, gameContexts := loadInputFiles(files, cfg.Devices.DeviceToShortNameMap,
		log, cfg.DebugOutput, cfg.VerboseOutput)
	common.GenerateContextColours(gameContexts, cfg)
	return sharedGameData, gameBinds, gameDevices, gameContexts, sharedGameData.Logo
}

// Load the game config files (provided by user)
func loadInputFiles(files [][]byte, deviceNameMap common.DeviceNameFullToShort,
	log *common.Logger, bool, verboseOutput bool) (common.GameBindsByProfile, common.Set,
	common.ContextToColours) {
	gameBindsByProfile := make(common.GameBindsByProfile)
	gameBinds := make(common.GameDeviceContextActions)
	gameBindsByProfile[common.ProfileDefault] = gameBinds
	deviceNames := make(common.Set)
	contexts := make(common.ContextToColours)

	// deviceIndex: deviceId -> full name
	deviceIndex := make(map[string]string)
	contextActionIndex := make(swsContextActionIndex)

	// Load all the device and inputs
	for idx, file := range files {
		scanner := scannerFactory(file)
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "GstKeyBinding.") {
				matches := sharedRegexes.Bind.FindStringSubmatch(line)
				if matches != nil {
					override, err := strconv.Atoi(matches[3])
					if err != nil {
						log.Err("SWS Device num not an integer %s", matches[3])
						continue
					}
					addAction(contextActionIndex, matches[1], contexts, matches[2],
						override, matches[4], matches[5])
					continue
				}
			} else if strings.HasPrefix(line, "GstInput.JoystickDevice") {
				matches2 := sharedRegexes.Joystick.FindStringSubmatch(line)
				if matches2 != nil && len(matches2[2]) > 0 {
					if shortName, found := deviceNameMap[matches2[2]]; !found {
						log.Err("SWS Unknown device found %s", matches2[2])
						continue
					} else {
						num, err := strconv.Atoi(matches2[1])
						// Subtract 1 from the Joystick index to match deviceIds in the file
						num--
						if err == nil && num >= 0 {
							deviceIndex[strconv.Itoa(num)] = shortName
							deviceNames[shortName] = true
						} else {
							log.Err("SWS unexpected device number %s", matches2[1])
						}
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
		for action, overrideActionSubMap := range actionMap {
			// Sort the override keys to ensure deterministic Primary/Secondary assignment
			var overrides []int
			for override := range overrideActionSubMap {
				overrides = append(overrides, override)
			}
			sort.Ints(overrides)

			// Don't need to use the override index
			for _, override := range overrides {
				actionSubMap := overrideActionSubMap[override]
				if actionSubMap["deviceid"] == "-1" {
					// Ignore deviceid -1
					continue
				}
				// Build the action details
				actionDetails := swsActionDetails{}
				for actionSub, value := range actionSubMap {
					field, err := getInputTypeAsField(actionSub, &actionDetails)
					if err != nil {
						log.Err("%s value %s", err, value)
					} else if field != nil { // Ignore nil fields, this info isn't needed
						*field = value
					}
				}

				shortName, found := deviceIndex[actionDetails.DeviceID]
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

				// Assign action details accordingly
				input, err := interpretInput(&actionDetails, shortName, context, action, log)
				if err != nil {
					log.Err("%s", err)
					continue
				}
				if len(input) > 0 {
					gameAction, found := actions[action]
					if !found {
						gameAction = make(common.GameInput, common.NumInputs)
						actions[action] = gameAction
						gameAction[common.InputPrimary] = input
					} else if gameAction[common.InputPrimary] != input {
						// Only add as secondary if its a different input.
						// You get duplication on Axis as there are separate Up/Down inputs
						// but the game config lists the same axis twice.
						gameAction[common.InputSecondary] = input
					}
				} else {
					// TODO - what's this for?
					delete(actions, action)
				}
			}
		}
	}

	return gameBindsByProfile, deviceNames, contexts
}

func addAction(contextActionIndex swsContextActionIndex, context string,
	contexts common.ContextToColours, action string, override int,
	actionSub string, value string) {
	contexts[context] = ""

	var found bool
	var actionMap map[string]map[int]map[string]string
	if actionMap, found = contextActionIndex[context]; !found {
		// First time for this context
		actionMap = make(map[string]map[int]map[string]string)
		contextActionIndex[context] = actionMap
	}
	var overrideActionSubMap map[int]map[string]string
	if overrideActionSubMap, found = actionMap[action]; !found {
		// First time for this device action sub map
		overrideActionSubMap = make(map[int]map[string]string)
		actionMap[action] = overrideActionSubMap
	}
	var actionSubMap map[string]string
	if actionSubMap, found = overrideActionSubMap[override]; !found {
		actionSubMap = make(map[string]string)
		overrideActionSubMap[override] = actionSubMap
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
	case "deviceid":
		return &currAction.DeviceID, nil
	case "altbutton", "identifier", "modifier", "negate", "type":
		// Don't need to store these but they aren't an error
		return nil, nil
	}
	// Unknown actionSub found
	return nil, fmt.Errorf("SWS unknown inputType %s", actionSub)
}

// interpretInput maps the game input to MRC's understanding of inputs.
// Returns a string that has the mapped value or an error.
// A mapped value of empty string with a nil error means ignore this
func interpretInput(details *swsActionDetails, device string, context string, action string,
	log *common.Logger) (string, error) {
	if details.DeviceID == "-1" {
		// Ignore inputs for deviceid -1. This is not an error
		return "", nil
	}
	// TODO - Currently hardcoded for the X-55 based on reverse engineering.
	switch details.Axis {
	case "8":
		return "XAxis", nil // Throttle
	case "9":
		return "YAxis", nil // Stick
	case "10":
		return "XAxis", nil // Stick
	case "11":
		return "YAxis", nil // Stick
	case "26":
		button, err := strconv.Atoi(details.Button)
		if err != nil {
			return "", fmt.Errorf("SWS button not number - device %s context %s action %s data %v",
				device, context, action, details)
		}
		if button > 21 && button < 40 {
			button -= 21 // Seems like a hardcoded number?
			return strconv.Itoa(button), nil
		} else if button >= 64 && button < 86 {
			button -= 45 // Another hardcoded number
			return strconv.Itoa(button), nil
		} else if button == 86 {
			return "", nil
		}
		switch device {
		case "SaitekX55Joystick":
			switch button {
			case 46:
				return "RZAxis", nil
			case 47:
				return "RZAxis", nil
			case 48:
				return "POV1Up", nil
			case 49:
				return "POV1Down", nil
			case 50:
				return "POV1Left", nil
			case 51:
				return "POV1Right", nil
			}
		case "SaitekX55Throttle":
			switch button {
			case 40:
				return "ZAxis", nil
			case 41:
				return "ZAxis", nil
			case 42:
				return "RXAxis", nil
			case 43:
				return "RXAxis", nil
			case 44:
				return "RYAxis", nil
			case 45:
				return "RYAxis", nil
			case 46:
				return "RZAxis", nil
			case 47:
				return "RZAxis", nil
			}
		}
	}
	return "", fmt.Errorf("SWS Unknown input - device %s context %s action %s data %v",
		device, context, action, details)

}

// matchGameInputToModel - returns a common.GameInput of the inputs that can be displayed.
// Also returns the label to use for error text
func matchGameInputToModel(deviceName string, gameInput common.GameInput,
	deviceInputs common.DeviceInputs, gameInputMap common.InputTypeMapping,
	log *common.Logger) (common.GameInput, string) {
	// For SWS, we've already got the structure right
	return gameInput, sharedGameData.Logo
}

// swsContextActionIndex: context -> action name -> override -> action sub -> value
type swsContextActionIndex map[string]map[string]map[int]map[string]string

type swsRegexes struct {
	Bind     *regexp.Regexp
	Joystick *regexp.Regexp
}

// Parsed fields from sws config
type swsActionDetails struct {
	// Unused  AltButton  string
	Axis     string
	Button   string
	DeviceID string
	// Unused  Identifier string
	// Unused  Modifier   string
	// Unused  Negate     string
	// Unused  Type       string
}

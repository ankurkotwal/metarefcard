package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/ankurkotwal/InputRefCard/data"
	"gopkg.in/yaml.v3"
)

func main() {
	debugOutput := false
	verboseOutput := false

	parseCliArgs(&debugOutput, &verboseOutput)

	// Load the game files provided
	gameDeviceNameMap := loadGameModel(debugOutput)
	gameBinds := loadGameConfigs(flag.Args(), gameDeviceNameMap, debugOutput, verboseOutput)

	neededDevices := make(map[string]bool)
	for device := range gameBinds {
		neededDevices[device] = true
	}
	// Load the abstract device model (i.e. non-game specific) based on the devices in our game files
	deviceIndex := data.LoadDeviceData(neededDevices, debugOutput)

	// Map the game input bindings to our model
	overlaysByImage := populateImageOverlays(deviceIndex, gameBinds)
	_ = overlaysByImage
}

func parseCliArgs(debugOutput *bool, verboseOutput *bool) {
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("file\tFlight Simulator 2020 input configration (XML).\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(debugOutput, "d", false, "Debug output.")
	flag.BoolVar(verboseOutput, "v", false, "Verbose output.")
	flag.Parse()
	args := flag.Args()
	if len(flag.Args()) < 1 {
		flag.Usage()
		print(args)
		os.Exit(1)
	}

}

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

// Load the game config files (provided by user)
func loadGameConfigs(files []string, deviceNameMap deviceNameFullToShort, debugOutput bool, verboseOutput bool) gameBindsByDevice {
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
								log.Printf("Error: Missing info for device '%s'", aDevice.DeviceName)
							}
						} else {
							deviceNameMap[shortName] = deviceUnknown
							log.Printf("Error: Unknown device '%s'", aDevice.DeviceName)
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
						log.Printf("Duplicate context: %s\n", contextName)
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
								log.Printf("Action %s flag parsing error", actionName)
							}
						}
					}
					var found bool
					currentAction, found = currentContext[actionName]
					if found {
						log.Printf("Duplicate action: %s\n", actionName)
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
						log.Printf("Primary key value %s  parsing error", value)
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

type fs2020Data struct {
	DeviceNameMap deviceNameFullToShort `yaml:"DeviceNameMap"`
}
type deviceNameFullToShort map[string]string

const (
	deviceUnknown     = "DeviceUnknown"     // Unfamiliar with this device
	deviceMissingInfo = "DeviceMissingInfo" // Only know the name of device
)

// Load FS2020 specific data from our model. Update the device names (map game device name to our model names)
func loadGameModel(debugOutput bool) deviceNameFullToShort {
	nameMap := fs2020Data{}
	// Load our game data
	yamlData, err := ioutil.ReadFile("data/fs2020.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	// Unmarshall yaml file to data structure
	err = yaml.Unmarshal([]byte(yamlData), &nameMap)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if debugOutput {
		d, err := yaml.Marshal(&nameMap)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("=== Device Name Map ===\n%s\n\n", string(d))
	}

	fullToShort := deviceNameFullToShort{}
	// Update map of game device names to our model device names
	for fullName, shortName := range nameMap.DeviceNameMap {
		if shortName != "" {
			fullToShort[fullName] = shortName
		} else {
			fullToShort[fullName] = deviceMissingInfo
		}
	}

	return fullToShort
}

func populateImageOverlays(deviceIndex data.DeviceModel, gameBinds gameBindsByDevice) data.OverlaysByImage {
	// Iterate through our game binds
	var overlaysByImage data.OverlaysByImage
	for deviceName, gameDevice := range gameBinds {
		modelDevice := deviceIndex[deviceName]
		image := modelDevice.Image
		for context, actions := range gameDevice.ContextActions {
			for actionName, actionData := range actions {
				inputDataList := findMatchingInputModels(actionData, *modelDevice.Inputs)
				for _, inputData := range inputDataList {
					var overlayData data.OverlayData
					overlayData.Context = context
					overlayData.Text = actionName
					overlayData.PosAndSize = &inputData

					if _, ok := overlaysByImage[image]; !ok {
						overlaysByImage[image] = make([]data.OverlayData, 0) // TODO don't hardcode 100
					}
					overlaysByImage[image] = append(overlaysByImage[deviceName], overlayData)
				}
			}
		}
	}

	return overlaysByImage
}

// FS2020 Device Name -> Index name, game inputs -> index inputs
// Joystick Button %d
// Button\d+
// Joystick L-Axis X/Y/Z -> Joy_XAxis
// Joystick R-Axis X/Y/Z -> JoyRXAxis, JoyRYAxis, JoyRZAxis
// Axis X/Y/Z (+/-)?
// Joystick Pov Up/Down/Left/Right -> Joy_POV1Up, Joy_POV1Down, Joy_POV1Left, Joy_POV1Right
// POV1_UP, POV1_DOWN, POV1_LEFT, POV1_RIGHT
// Rotation X/Y/Z (+/-)?
// Slider X/Y (+/-)?

// Takes the game provided bindings with the internal device map to
// build a list of image overlays.
func findMatchingInputModels(actionData *gameAction, inputs data.InputsMap) []data.InputData {
	inputDataList := make([]data.InputData, 0)

	// TODO - avoid repeated declaration of regexs
	buttonRegex := regexp.MustCompile(`Button\s*(\d+)`)
	axisRegex1 := regexp.MustCompile(`Axis\s*([XYZ])\s*([+-])?`)
	axisRegex2 := regexp.MustCompile(`(?:([LR])-)Axis\s*([XYZ])\s*([+-])?`)
	povRegex := regexp.MustCompile(`(?i)Pov[\s_]([[:alpha:]]+)`)
	rotationRegex := regexp.MustCompile(`Rotation\s*([XYZ])\s*([+-])?`)
	sliderRegex := regexp.MustCompile(`Slider\s*([XYZ])\s*([+-])?`)

	matches := buttonRegex.FindAllStringSubmatch((*actionData).PrimaryInfo, -1)
	if matches != nil {
	} else {
		matches = axisRegex1.FindAllStringSubmatch((*actionData).PrimaryInfo, -1)
		if matches != nil {
		} else {
			matches = axisRegex2.FindAllStringSubmatch((*actionData).PrimaryInfo, -1)
			if matches != nil {
			} else {
				matches = povRegex.FindAllStringSubmatch((*actionData).PrimaryInfo, -1)
				if matches != nil {
				} else {
					matches = rotationRegex.FindAllStringSubmatch((*actionData).PrimaryInfo, -1)
					if matches != nil {
					} else {
						matches = sliderRegex.FindAllStringSubmatch((*actionData).PrimaryInfo, -1)
						if matches != nil {
						} else {
							log.Printf("Error: Could not find matching Action %v", *actionData)
						}
					}
				}
			}
		}
	}
	_ = matches

	_ = buttonRegex
	_ = axisRegex1
	_ = axisRegex2
	_ = povRegex
	_ = rotationRegex
	_ = sliderRegex

	return inputDataList
}

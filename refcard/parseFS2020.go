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
	gameBinds, gameDeviceNames := loadGameConfigs(flag.Args(), debugOutput, verboseOutput)
	loadGameData(gameDeviceNames, debugOutput)

	// Create a mapping of game device names to short names from our model
	gameDeviceShortNames := make(map[string]bool)
	for _, shortName := range *gameDeviceNames {
		gameDeviceShortNames[shortName] = true
	}

	// Load the abstract device model (i.e. non-game specific) based on the devices in our game files
	deviceIndex := data.LoadDeviceData(&gameDeviceShortNames, debugOutput)

	// Map the game input bindings to our model
	mapGameBindsToIndex(gameBinds, deviceIndex)
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

const (
	keyUnknown   = iota
	keyPrimary   = iota
	keySecondary = iota
)

// Load the game config files (provided by user)
func loadGameConfigs(files []string, debugOutput bool, verboseOutput bool) (*gameBindsByDevice, *map[string]string) {
	gameBinds := make(gameBindsByDevice)
	gameDevices := make(map[string]string)

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
						gameBinds[aDevice.DeviceName] = currentDevice
						gameDevices[aDevice.DeviceName] = deviceUnknown
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
	return &gameBinds, &gameDevices
}

type fs2020Data struct {
	DeviceNameMap map[string]string `yaml:"DeviceNameMap"`
}

const (
	deviceUnknown     = "DeviceUnknown"     // Unfamiliar with this device
	deviceMissingInfo = "DeviceMissingInfo" // Only know the name of device
)

// Load FS2020 specific data from our model. Update the device names (map game device name to our model names)
func loadGameData(gameDeviceNames *map[string]string, debugOutput bool) {
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

	// Update map of game device names to our model device names
	for fullName := range *gameDeviceNames {
		if shortName, found := nameMap.DeviceNameMap[fullName]; found {
			if len(shortName) == 0 {
				(*gameDeviceNames)[fullName] = deviceMissingInfo // Know of the device but not much else
			} else {
				(*gameDeviceNames)[fullName] = shortName
			}
		} else {
			(*gameDeviceNames)[fullName] = deviceUnknown // Don't know anything about this device
		}

	}
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

type imageOverlay struct {
	Text      string
	InputData *gameAction
}
type overlaysByImage map[string]imageOverlay

func mapGameBindsToIndex(gameBinds *gameBindsByDevice, deviceIndex *data.DeviceMap) *overlaysByImage {
	overlaysByImage := overlaysByImage{}
	buttonRegex := regexp.MustCompile(`Button\s*(\d+)`)
	axisRegex1 := regexp.MustCompile(`Axis\s*([XYZ])\s*([+-])?`)
	axisRegex2 := regexp.MustCompile(`(?:([LR])-)Axis\s*([XYZ])\s*([+-])?`)
	povRegex := regexp.MustCompile(`(?i)Pov[\s_]([[:alpha:]])`)
	rotationRegex := regexp.MustCompile(`Rotation\s*([XYZ])\s*([+-])?`)
	sliderRegex := regexp.MustCompile(`Slider\s*([XYZ])\s*([+-])?`)

	_ = overlaysByImage
	_ = buttonRegex
	_ = axisRegex1
	_ = axisRegex2
	_ = povRegex
	_ = rotationRegex
	_ = sliderRegex

	return &overlaysByImage
}

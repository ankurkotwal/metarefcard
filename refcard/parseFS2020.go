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
	"strings"

	"github.com/ankurkotwal/InputRefCard/data"
	"github.com/fogleman/gg"
	"gopkg.in/yaml.v3"
)

func main() {
	debugOutput := false
	verboseOutput := false

	parseCliArgs(&debugOutput, &verboseOutput)

	// Load the game files provided
	gameDeviceNameMap, sliderInputMap := loadGameModel(debugOutput)
	gameBinds := loadGameConfigs(flag.Args(), gameDeviceNameMap,
		debugOutput, verboseOutput)

	neededDevices := make(map[string]bool)
	for device := range gameBinds {
		neededDevices[device] = true
	}
	// Load the abstract device model (i.e. non-game specific) based on the devices in our game files
	deviceIndex := data.LoadDeviceData(neededDevices, debugOutput)

	// Setup the regexes once and pass them around
	regexes := make(map[string]*regexp.Regexp)
	regexes["button"] = regexp.MustCompile(`Button\s*(\d+)`)
	regexes["axis"] = regexp.MustCompile(`(?:([LR])-)?Axis\s*([XYZ])?\s*([+-])?`)
	regexes["pov"] = regexp.MustCompile(`(?i)Pov(\d?)[\s_]([[:alpha:]]+)`)
	regexes["rotation"] = regexp.MustCompile(`Rotation\s*([XYZ])\s*([+-])?`)
	regexes["slider"] = regexp.MustCompile(`Slider\s*([XYZ])\s*([+-])?`)

	// Map the game input bindings to our model
	overlaysByImage := populateImageOverlays(deviceIndex, gameBinds, regexes, sliderInputMap)
	for imageFilename, overlayDataRange := range overlaysByImage {
		image, err := gg.LoadImage(imageFilename)
		_ = image
		if err != nil {
			log.Printf("Error: loadImage %s failed. %v\n", imageFilename, err)
			continue
		}
		for _, overlayData := range overlayDataRange {
			dc := gg.NewContext(overlayData.PosAndSize.Width, overlayData.PosAndSize.Height)
			dc.SetRGBA(1, 1, 1, 1)

		}
	}
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

type fs2020Data struct {
	DeviceNameMap  deviceNameFullToShort `yaml:"DeviceNameMap"`
	SliderInputMap sliderInputMap        `yaml:"SliderInputMapping"`
}
type deviceNameFullToShort map[string]string
type sliderInputMap map[string]string // Device name -> model input

const (
	deviceUnknown     = "DeviceUnknown"     // Unfamiliar with this device
	deviceMissingInfo = "DeviceMissingInfo" // Only know the name of device
)

// Load FS2020 specific data from our model. Update the device names (map game device name to our model names)
func loadGameModel(debugOutput bool) (deviceNameFullToShort, sliderInputMap) {
	data := fs2020Data{}
	// Load our game data
	yamlData, err := ioutil.ReadFile("data/fs2020.yaml")
	if err != nil {
		log.Printf("Error: yamlFile.Get err   #%v\n", err)
	}
	// Unmarshall yaml file to data structure
	err = yaml.Unmarshal([]byte(yamlData), &data)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if debugOutput {
		d, err := yaml.Marshal(&data)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("=== FS2020 Data ===\n%s\n\n", string(d))
	}

	fullToShort := deviceNameFullToShort{}
	// Update map of game device names to our model device names
	for fullName, shortName := range data.DeviceNameMap {
		if shortName != "" {
			fullToShort[fullName] = shortName
		} else {
			fullToShort[fullName] = deviceMissingInfo
		}
	}

	return fullToShort, data.SliderInputMap
}

func populateImageOverlays(deviceIndex data.DeviceModel, gameBinds gameBindsByDevice,
	regexes map[string]*regexp.Regexp, sliderInputMap sliderInputMap) data.OverlaysByImage {
	// Iterate through our game binds
	overlaysByImage := make(data.OverlaysByImage)
	for deviceName, gameDevice := range gameBinds {
		modelDevice := deviceIndex[deviceName]
		image := modelDevice.Image
		for context, actions := range gameDevice.ContextActions {
			for actionName, actionData := range actions {
				inputDataList := findMatchingInputModels(deviceName, actionData, *modelDevice.Inputs, regexes, sliderInputMap)
				for _, inputData := range inputDataList {
					var overlayData data.OverlayData
					overlayData.Context = context
					overlayData.Text = actionName
					overlayData.PosAndSize = &inputData

					if _, ok := overlaysByImage[image]; !ok {
						overlaysByImage[image] = make([]data.OverlayData, 0)
					}
					overlaysByImage[image] = append(overlaysByImage[image], overlayData)
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
func findMatchingInputModels(deviceName string, actionData *gameAction, inputs data.InputsMap,
	regexes map[string]*regexp.Regexp, sliderInputMap sliderInputMap) []data.InputData {
	inputDataList := make([]data.InputData, 0)

	var inputData *data.InputData
	inputData = findMatchingInputModelsInner(deviceName, (*actionData).PrimaryInfo, inputs, regexes, sliderInputMap)
	if inputData != nil {
		inputDataList = append(inputDataList, *inputData)
	} else {
		log.Printf("Error: Did not find primary input for %s\n", (*actionData).PrimaryInfo)
	}
	if len((*actionData).SecondaryInfo) > 0 {
		inputData = findMatchingInputModelsInner(deviceName, (*actionData).SecondaryInfo, inputs, regexes, sliderInputMap)
		if inputData != nil {
			inputDataList = append(inputDataList, *inputData)
		} else {
			log.Printf("Error: Did not find secondary input for %s\n", (*actionData).SecondaryInfo)
		}
	}
	return inputDataList
}

func findMatchingInputModelsInner(deviceName string, action string, inputs data.InputsMap,
	regexes map[string]*regexp.Regexp, sliderInputMap sliderInputMap) *data.InputData {
	var matches [][]string

	matches = regexes["button"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		button := fmt.Sprintf("Joy_%s", matches[0][1])
		modelInput := inputs[button]
		return &modelInput
	}

	matches = regexes["axis"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		// TODO Axis
		axis := fmt.Sprintf("Joy_%s", matches[0][1])
		modelInput := inputs[axis]
		return &modelInput
	}
	matches = regexes["pov"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		direction := strings.Title(strings.ToLower(matches[0][2]))
		pov := fmt.Sprintf("Joy_POV%s%s", "1", direction)
		if len(matches[0][1]) > 0 {
			pov = fmt.Sprintf("Joy_POV%s%s", matches[0][1], direction)
		}
		modelInput := inputs[pov]
		return &modelInput
	}

	matches = regexes["rotation"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		rotation := fmt.Sprintf("Joy_R%sAxis", matches[0][1])
		modelInput := inputs[rotation]
		return &modelInput
	}

	matches = regexes["slider"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		var slider string
		if input, ok := sliderInputMap[deviceName]; ok {
			slider = input
		} else {
			log.Printf("Error: Unknown action %s for slider on device %s\n", action, deviceName)
			return nil
		}
		if modelInput, ok := inputs[slider]; ok {
			return &modelInput
		}
		log.Printf("Error: Couldn't find slider %s on device %s\n", slider, deviceName)
		return nil
	}

	log.Printf("Error: Could not find matching Action %s on device %s\n", action, deviceName)
	return nil
}

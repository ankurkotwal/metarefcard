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

const appName string = "MetaRef"
const appVersion string = "alpha"

func main() {
	debugOutput := false
	verboseOutput := false

	parseCliArgs(&debugOutput, &verboseOutput)

	// Load the game files provided
	gameData := loadGameModel(debugOutput)
	gameBinds := loadGameConfigs(flag.Args(), gameData.DeviceNameMap,
		debugOutput, verboseOutput)

	neededDevices := make(map[string]bool)
	for device := range gameBinds {
		neededDevices[device] = true
	}
	// Load the abstract device model (i.e. non-game specific) based on the devices in our game files
	deviceIndex := data.LoadDeviceData(neededDevices, debugOutput)
	// Add device additions to the main device index
	for deviceName, deviceInputData := range gameData.InputAdditions {
		if deviceData, found := deviceIndex[deviceName]; found {
			for additionInput, additionData := range deviceInputData.Inputs {
				deviceData.Inputs[additionInput] = additionData
			}
		}
	}

	// Setup the regexes once and pass them around
	regexes := make(map[string]*regexp.Regexp)
	regexes["button"] = regexp.MustCompile(`Button\s*(\d+)`)
	regexes["axis"] = regexp.MustCompile(`(?:([R])-)?Axis\s*([XYZ])`)
	regexes["pov"] = regexp.MustCompile(`(?i)Pov(\d?)[\s_]([[:alpha:]]+)`)
	regexes["rotation"] = regexp.MustCompile(`Rotation\s*([XYZ])`)
	regexes["slider"] = regexp.MustCompile(`Slider\s*([XYZ])`)

	// TODO different Font sizes
	const fontSize int = 40
	const pixelInset int = 10
	fontFace, err := gg.LoadFontFace("resources/fonts/Roboto-Regular.ttf", float64(fontSize))
	if err != nil {
		panic(err)
	}

	// Map the game input bindings to our model
	overlaysByImage := populateImageOverlays(deviceIndex, gameBinds, regexes, gameData)
	for imageFilename, overlayDataRange := range overlaysByImage {
		image, err := gg.LoadImage(fmt.Sprintf("resources/hotas_images/%s", imageFilename))
		if err != nil {
			log.Printf("Error: loadImage %s failed. %v\n", imageFilename, err)
			continue
		}
		dc := gg.NewContextForImage(image)
		dc.SetRGB(0, 0, 0)
		dc.SetFontFace(fontFace)
		for _, overlayData := range overlayDataRange {
			dc.DrawString(overlayData.Text,
				float64(overlayData.PosAndSize.ImageX+pixelInset),
				float64(overlayData.PosAndSize.ImageY+fontSize))
		}
		_ = os.Mkdir("out", os.ModePerm)
		dc.SavePNG(fmt.Sprintf("out/%s", imageFilename))
	}
	fmt.Println("Done")
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
	DeviceNameMap  deviceNameFullToShort           `yaml:"DeviceNameMap"`
	InputMap       deviceInputTypeMapping          `yaml:"InputMapping"`
	InputAdditions map[string]data.DeviceInputData `yaml:"InputAdditions"`
	InputLabels    map[string]string               `yaml:"InputLabels"`
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

// Load FS2020 specific data from our model. Update the device names (map game device name to our model names)
func loadGameModel(debugOutput bool) *fs2020Data {
	data := fs2020Data{}
	// Load our game data
	yamlData, err := ioutil.ReadFile("data/fs2020.yaml")
	if err != nil {
		log.Printf("Error: yamlFile.Get err   #%v\n", err)
	}
	// Unmarshall yaml file to data structure
	err = yaml.Unmarshal([]byte(yamlData), &data)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	if debugOutput {
		d, err := yaml.Marshal(&data)
		if err != nil {
			log.Fatalf("Error: %v", err)
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
	data.DeviceNameMap = fullToShort

	return &data
}

func populateImageOverlays(deviceIndex data.DeviceModel, gameBinds gameBindsByDevice,
	regexes map[string]*regexp.Regexp, gameData *fs2020Data) data.OverlaysByImage {
	// Iterate through our game binds
	overlaysByImage := make(data.OverlaysByImage)
	for deviceName, gameDevice := range gameBinds {
		modelDevice := deviceIndex[deviceName]
		image := modelDevice.Image
		for context, actions := range gameDevice.ContextActions {
			for actionName, actionData := range actions {
				inputLookups := findMatchingInputModels(deviceName, actionData,
					modelDevice.Inputs, regexes, (*gameData).InputMap[deviceName])
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

// Takes the game provided bindings with the internal device map to
// build a list of image overlays.
func findMatchingInputModels(deviceName string, actionData *gameAction, inputs data.InputsMap,
	regexes map[string]*regexp.Regexp, gameInputMap inputTypeMapping) []string {
	inputLookups := make([]string, 0)

	input := findMatchingInputModelsInner(deviceName,
		(*actionData).PrimaryInfo, inputs, regexes, gameInputMap)
	if input != "" {
		inputLookups = append(inputLookups, input)
	} else {
		log.Printf("Error: Did not find primary input for %s\n", (*actionData).PrimaryInfo)
	}
	if len((*actionData).SecondaryInfo) > 0 {
		input := findMatchingInputModelsInner(deviceName,
			(*actionData).SecondaryInfo, inputs, regexes, gameInputMap)
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
	regexes map[string]*regexp.Regexp, gameInputMap inputTypeMapping) string {
	var matches [][]string

	matches = regexes["button"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		return matches[0][1]
	}

	matches = regexes["axis"].FindAllStringSubmatch(action, -1)
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
	matches = regexes["pov"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		direction := strings.Title(strings.ToLower(matches[0][2]))
		pov := fmt.Sprintf("POV%s%s", "1", direction)
		if len(matches[0][1]) > 0 {
			pov = fmt.Sprintf("POV%s%s", matches[0][1], direction)
		}
		return pov
	}

	matches = regexes["rotation"].FindAllStringSubmatch(action, -1)
	if matches != nil {
		rotation := fmt.Sprintf("R%sAxis", matches[0][1])
		return rotation
	}

	matches = regexes["slider"].FindAllStringSubmatch(action, -1)
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

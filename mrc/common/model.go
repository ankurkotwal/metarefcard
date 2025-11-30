package common

import (
	"fmt"
	"sort"
)

// LoadGameModel - load game specific data from our model. Update the device names
// (map game device name to our model names)
func LoadGameModel(filename string, label string, debugOutput bool, log *Logger) GameData {
	data := GameData{}
	LoadYaml(filename, &data, label, log)
	return data
}

// FilterDevices - Returns only the devices that the caller is asking for
func FilterDevices(neededDevices Set, config *Config, log *Logger) DeviceMap {
	filteredDevices := make(DeviceMap)
	// Filter for only the device groups we're interested in
	for shortName := range neededDevices {
		if _, found := config.Devices.Index[shortName]; !found {
			log.Err("device not found in config %s", shortName)
			continue
		}
		neededDevices[shortName] = true
	}
	for shortName, inputs := range config.Devices.Index {
		if _, found := neededDevices[shortName]; found {
			filteredDevices[shortName] = inputs
		}
	}

	if config.DebugOutput {
		log.Dbg("%s", YamlObjectAsString(filteredDevices, "Targeted Device Map"))
	}
	return filteredDevices
}

// GenerateContextColours - basic utility function to generate colours
func GenerateContextColours(contexts ContextToColours, config *Config) {
	contextKeys := contexts.Keys()
	sort.Strings(contextKeys)
	i := 0
	for _, context := range contextKeys {
		if i >= len(config.AlternateColours) {
			// Ran out of colours, repeat
			i = 0
		}
		// Only move to next colour if this is an unseen category
		contexts[context] = config.AlternateColours[i]
		i++
	}
}

// FuncRequestHandler - handles incoming requests and returns game data, game binds,
// neededDevices and a context to colour mapping
type FuncRequestHandler func(files [][]byte, config *Config, log *Logger) (GameData,
	GameBindsByProfile, Set, ContextToColours, string)

// FuncMatchGameInputToModel takes the game provided bindings with the device map to
// build a list of image overlays.
type FuncMatchGameInputToModel func(deviceName string, actionData GameInput,
	deviceInputs DeviceInputs, gameInputMap InputTypeMapping, log *Logger) (GameInput, string)

// PopulateImageOverlays returns a list of image overlays to put on device images
func PopulateImageOverlays(neededDevices Set, config *Config, log *Logger,
	gameBindsByProfile GameBindsByProfile, gameData GameData, matchFunc FuncMatchGameInputToModel) OverlaysByProfile {

	deviceMap := FilterDevices(neededDevices, config, log)
	imageMap := config.Devices.ImageMap

	// Iterate through our game binds
	overlaysByProfile := make(OverlaysByProfile)
	for profile, gameBinds := range gameBindsByProfile {
		overlaysByImage := make(OverlaysByImage)
		overlaysByProfile[profile] = overlaysByImage
		for shortName, gameDevice := range gameBinds {
			inputs := deviceMap[shortName]
			image := imageMap[shortName]
			for context, actions := range gameDevice {
				for actionName, gameInput := range actions {

					inputLookups, label := matchFunc(shortName, gameInput, inputs,
						gameData.InputMap[shortName], log)

					for idx, input := range inputLookups {
						if idx != 0 && len(input) == 0 {
							// Ok to have no input if its not the primary input
							// i.e. Might not have a secondary input
							continue
						}

						inputData, found := inputs[input]
						if !found {
							log.Err("%s unknown input to lookup %s for device %s",
								label, input, shortName)
						}
						if inputData.X == 0 && inputData.Y == 0 {
							log.Err("%s location 0,0 for %s device %s %v",
								label, actionName, shortName, inputData)
							continue
						}
						GenerateImageOverlays(overlaysByImage, input, inputData,
							gameData, actionName, context, shortName, image, label, log)
					}
				}
			}
		}
	}

	return overlaysByProfile
}

// GenerateImageOverlays - creates the image overlays into overlaysByImage
func GenerateImageOverlays(overlaysByImage OverlaysByImage, input string, inputData InputData,
	gameData GameData, actionName string, context string, shortName string,
	image string, gameLabel string, log *Logger) {
	var overlayData OverlayData
	overlayData.ContextToTexts = make(map[string][]string)
	overlayData.PosAndSize = inputData
	var text string
	// Game data might have a better label for this text
	if label, found := gameData.InputLabels[actionName]; found {
		text = label
	} else {
		text = actionName
		log.Err("%s label not found. %s context %s device %s",
			gameLabel, actionName, context, shortName)
	}
	texts := make([]string, 1)
	texts[0] = text
	overlayData.ContextToTexts[context] = texts

	// Find by Image first
	deviceAndInput := fmt.Sprintf("%s:%s", shortName, input)
	if overlay, found := overlaysByImage[image]; !found {
		// First time adding this image
		overlay := make(map[string]OverlayData)
		overlaysByImage[image] = overlay
		overlay[deviceAndInput] = overlayData
	} else {
		// Now find by input
		if previousOverlayData, found := overlay[deviceAndInput]; !found {
			// Not new image but new overlayData
			overlay[deviceAndInput] = overlayData
		} else {
			// Concatenate input
			texts = append(previousOverlayData.ContextToTexts[context], text)
			sort.Strings(texts)
			previousOverlayData.ContextToTexts[context] = texts
		}
	}
}

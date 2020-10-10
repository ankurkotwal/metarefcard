package common

import (
	"fmt"
	"log"
	"sort"
)

// LoadGameModel - load game specific data from our model. Update the device names
// (map game device name to our model names)
func LoadGameModel(filename string, label string, debugOutput bool) *GameData {
	data := GameData{}
	LoadYaml(filename, &data, label)

	fullToShort := DeviceNameFullToShort{}
	// Update map of game device names to our model device names
	for fullName, shortName := range data.DeviceNameMap {
		if shortName != "" {
			fullToShort[fullName] = shortName
		} else {
			fullToShort[fullName] = DeviceMissingInfo
		}
	}
	data.DeviceNameMap = fullToShort

	return &data
}

// FilterDevices - Returns only the devices that the caller is asking for
func FilterDevices(neededDevices MockSet, config *Config) DeviceMap {
	filteredDevices := make(DeviceMap)
	// Filter for only the device groups we're interested in
	for shortName := range neededDevices {
		if _, found := config.DeviceMap[shortName]; !found {
			log.Printf("Error: device not found in config %s", shortName)
			continue
		}
		neededDevices[shortName] = ""
	}
	for shortName, inputs := range config.DeviceMap {
		if _, found := neededDevices[shortName]; found {
			filteredDevices[shortName] = inputs
		}
	}

	if config.DebugOutput {
		PrintYamlObject(filteredDevices, "Targeted Device Map")
	}
	return filteredDevices
}

// GenerateContextColours - basic utility function to generate colours
func GenerateContextColours(contexts MockSet, config *Config) {
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

func GenerateImageOverlays(overlaysByImage OverlaysByImage, input string, inputData *InputData,
	gameData *GameData, actionName string, context string, shortName string,
	image string) {
	var overlayData OverlayData
	overlayData.ContextToTexts = make(map[string][]string)
	overlayData.PosAndSize = inputData
	var text string
	// Game data might have a better label for this text
	if label, found := (*gameData).InputLabels[actionName]; found {
		text = label
	} else {
		text = actionName
		log.Printf("Error: FS2020 Unknown action %s context %s device %s\n",
			actionName, context, shortName)
	}
	texts := make([]string, 1)
	texts[0] = text
	overlayData.ContextToTexts[context] = texts

	// Find by Image first
	deviceAndInput := fmt.Sprintf("%s:%s", shortName, input)
	if overlay, found := overlaysByImage[image]; !found {
		// First time adding this image
		overlay := make(map[string]*OverlayData)
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

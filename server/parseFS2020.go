package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type device struct {
	DeviceName string
	GUID       string
	ProductID  string
	Contexts   map[string]context
}
type context struct {
	ContextName string
	Actions     []action
}
type action struct {
	ActionName    string
	Flag          int
	PrimaryInfo   string
	PrimaryKey    int
	SecondaryInfo string
	SecondaryKey  int
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("file\tFlight Simulator 2020 input configration (XML).\n")
		flag.PrintDefaults()
	}
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	devicesByName := make(map[string]device)
	for _, filename := range os.Args[1:] {
		log.Printf("Opening file %s\n", filename)
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
				if ty.Name.Local == "Device" {
					var aDevice device
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
					device, found := devicesByName[aDevice.DeviceName]
					if found {
						out, _ := json.Marshal(device)
						log.Printf("Found existing device: %s\n", out)
						device.DeviceName = aDevice.DeviceName
						device.GUID = aDevice.GUID
						device.ProductID = aDevice.ProductID
					} else {
						device = aDevice
						out, _ := json.Marshal(device)
						log.Printf("Found new device: %s\n", out)
					}
				} else if ty.Name.Local == "Context" {
				} else if ty.Name.Local == "Action" {
				} else if ty.Name.Local == "Primary" {
				} else if ty.Name.Local == "KEY" {
				}
			default:
			}
		}
	}
}

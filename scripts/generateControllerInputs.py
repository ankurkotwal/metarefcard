#!/usr/bin/env python3

import sys
sys.path.append("../3rdparty/edrefcard")

from bindingsData import *

outFilename = '../refcard/data/generatedDevices.go'

output = []

output.append('''package data

// BuildIndex - builds the device index
func BuildIndex() DeviceIndexByGroupName {
    var deviceIndex DeviceIndexByGroupName
    var deviceGroup *DeviceGroup
    var deviceData *DeviceData
    var inputDataByName InputDataByName
    var inputData *InputData

    deviceIndex = make(DeviceIndexByGroupName)

''')

for name in supportedDevices :
    group = supportedDevices[name]
    image = group['Template']
    devices = group['HandledDevices']
    output.append('    //#region {n}\n'.format(n=name))
    output.append('    deviceGroup = new(DeviceGroup)\n')
    output.append('    deviceIndex["{n}"] = deviceGroup\n'.format(n=name))
    output.append('    deviceGroup.Image = "{i}.jpg"\n'.format(i=image))
    output.append('    deviceGroup.Devices = make(DevicesByName)\n')
    for device in devices:
        if device == 'Keyboard':
            continue
        output.append('\n')
        output.append('    deviceData = new(DeviceData)\n')
        output.append('    deviceGroup.Devices["{n}"] = deviceData\n'.format(n=device))
        output.append('    inputDataByName = make(InputDataByName)\n')
        output.append('    deviceData.DisplayName = "{d}"\n'.format(d=device))
        output.append('    deviceData.InputDataByName = &inputDataByName\n')
        
        for key in hotasDetails[device]:
            data = hotasDetails[device][key]
            if key == 'displayName':
                output.append('    deviceData.DisplayName = "{dn}"\n'.format(dn=data))
            else:
                output.append('    inputData = new(InputData)\n'.format())
                if data['Type'] == 'Digital':
                    output.append('    inputData.IsDigital = true\n')
                else:
                    output.append('    inputData.IsDigital = false\n')
                output.append('    inputData.ImageX = {x}\n'.format(x=data['x']))
                output.append('    inputData.ImageX = {x}\n'.format(x=data['x']))
                output.append('    inputData.ImageWidth = {w}\n'.format(w=data['width']))
                if 'height' in data:
                    output.append('    inputData.ImageHeight = {h}\n'.format(h=data['height']))
                else:
                    output.append('    inputData.ImageHeight = 54\n')
                output.append('    inputDataByName["{k}"] = inputData\n'.format(k=key))
    output.append('    //#endregion\n')

output.append('\n')
output.append('''return deviceIndex
}''')

outFile = open(outFilename, "w")
for line in output:
    print(line)
    outFile.write(line)
outFile.close()
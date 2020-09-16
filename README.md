# HotasRefCard

# Plan
1. ~~Read FS2020 xml files~~
2. ~~Read the EDRefCard inputs~~
3. ~~Build a model of game inputs and controller mappings~~
4. ~~Generate images~~
5. ~~Dynamic font size~~
6. Regenerate hotas_images, new X55 locations, vkb-kosmosima-scg-left 3879x2182, x-45 5120x2880, 
7. Sliders
8. Keyboard & mouse
9. Host on the web
10. Text wrapping
11. Extend to Elite Dangerous


# Setup

## Python
### PyYaml
Install modules
```pip3 install pyyaml```

# Generate Device Model
Read `3rdparty/edrefcard/bindingsData.py` to generate a custom configuration.
Command:
```generateControllerInputs.py```

# Generate Hotas Images
Generate jpgs of the Hotas images found in `assets/hotas_images` into `refcard/resources/hotas_images`
Command:
```generateHotasImages.py```

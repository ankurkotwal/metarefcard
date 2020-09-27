# HotasRefCard

# Plan
1. ~~Read FS2020 xml files~~
2. ~~Read the EDRefCard inputs~~
3. ~~Build a model of game inputs and controller mappings~~
4. ~~Generate images~~
5. ~~Dynamic font size~~
6. ~~Regenerate hotas_images, new X55 locations, vkb-kosmosima-scg-left 3879x2182, x-45 5120x2880~~
7. ~~Convert to webapp~~
8. ~~Sliders~~
9.  ~~Add game banner~~
10. ~~Colours~~
11. ~~Make images clickable to open a new tab~~
12. ~~Build container image~~
13. ~~Publish on Cloud Run~~
14. ~~Add Google Analytics~~
15. Add a message for unsupported devices. Maybe capture those configs to process later?
16. Add early testing info text, Github repo, how to report an issue
17. Watermark
18. Generate game name + device name
19. Keyboard & mouse
20. Extend to Elite Dangerous


# Setup

## Python
### Generate Device Model
#### Dependencies
Install modules
```pip3 install pyyaml```
#### Running the script
Read `3rdparty/edrefcard/bindingsData.py` to generate a custom configuration.
Command:
```generateControllerInputs.py```

# Generate Hotas Images
#### Dependencies
* Inkscape
* Imagemagick
```pip3 install ```
#### Running the script
Generate jpgs of the Hotas images found in `assets/hotas_images` into `refcard/resources/hotas_images`
Command:
```generateHotasImages.py```

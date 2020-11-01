# MetaRefCard (MRC)
A tool to generate reference cards for HOTAS and game controllers. MRC is
designed to be flexible across many different controllers and many different
games.


# Setup
### Generate Device Model
Generates a yaml file based on `resources-source/edrefcard/bindingsData.py` that MRC can read.
#### Dependencies
Install modules - `pip3 install pyyaml`
#### Running the script
Command: `generateControllerInputs.py`

# Generate Hotas & Logo Images
Convert and resize source resources into *configured* sizes for MetaRefCard. Source images are found in `resources-source/hotas-images` and `resources-source/game-logos`. They exported to `resources/hotas_images` and `resources/game_logos` respectively.
#### Dependencies
* Inkscape - `sudo apt install inkscape`
* Imagemagick - `sudo apt install imagemagick`
#### Running the script
Command: `generateHotasImages.py`

# Code
MRC is written in Go and is a web application.
# Packages
The entry package is `metarefcard`. Within this package is another package called `common` as well as a package for  each game that is supported. For example Flight Simulator 2020 is under `fs2020` and Star Wars: Squadrons is under `sws`. `common` contains code that is shared across all the game packages.
# Directories
`config` - runtime configuration. `config.yaml` is the main configuration file. Each package has their own config files too.
`metarefcard` - almost all of the go code for MRC.
`resources` - runtime resource files like fonts, web templates, static web files, game logos and generated hotas images.
`resources-source-source` - contains the source files used to generate resources
`testdata` - sample game input files used for testing.
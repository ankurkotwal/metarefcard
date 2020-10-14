#!/usr/bin/env python3

import os
import subprocess
import yaml

DEBUG_OUTPUT = False


def initialise():
    # Load configuration
    config = {}
    with open("config/config.yaml", "r") as stream:
        try:
            config = yaml.safe_load(stream)
        except yaml.YAMLError as exc:
            print("Error loading config.yaml {e}".format(e=exc))
    if DEBUG_OUTPUT:
        print("Config loaded: {c}\n".format(c=config))

    # Find inkscape binary
    inkscape = subprocess.run(
        ["which", "inkscape"], capture_output=True).stdout.decode("utf-8").rstrip("\n")
    if False == os.path.isfile(inkscape):
        print("Inkscape not found in path")
        exit(2)
    if DEBUG_OUTPUT:
        print("Found Inkscape at {i}".format(i=inkscape))

    dirs_images = ["assets/game_logos", "assets/hotas_images"]
    for dir in dirs_images:
        # Confirm source images exist
        if False == os.path.isdir(dir):
            print("Error could not find dir {d}".format(d=dir))
            exit(1)
        if DEBUG_OUTPUT:
            print("Found hotas images at {d}".format(d=dir))

    # Confirm destination directory
    dirs_out = [config["LogoImagesDir"], config["HotasImagesDir"]]
    for dir_out in dirs_images:
        if False == os.path.isdir(dir_out):
            os.mkdir(dir_out)
            # Check that mkdir succeeded
            if False == os.path.isdir(dir_out):
                print("mkdir {d} failed".format(d=dir_out))
                exit(3)
        if DEBUG_OUTPUT:
            print("Found destination dir {d}".format(d=dir_out))
    return dirs_images, dirs_out, inkscape, config


def convertfile(inkscape, svg, defaultwidth, defaultheight, multiplier,
                overrides, dir_out):
    name = os.path.splitext(os.path.basename(svg))[0]
    out = "{dir}/{out}.png".format(dir=dir_out, out=name)

    # Calculate new resolution
    width = defaultwidth * multiplier
    height = defaultheight * multiplier
    if name in overrides:
        width = int(overrides[name]["h"]) * multiplier
        height = int(overrides[name]["h"]) * multiplier

    # Convert svg to png with Inkscape
    cmd_export = [inkscape,
                  "--export-png={o}".format(o=out),
                  "-w={w}".format(w=width),
                  "-h={h}".format(h=height),
                  svg]
    convert = subprocess.run(cmd_export,
                             stderr=subprocess.PIPE,
                             stdout=subprocess.PIPE)
    if convert.returncode != 0:
        print("Error: Failed to convert {f}".format(f=name))
    else:
        print("Converted {f}".format(f=name))


def main():
    dirs_images, dirs_out, inkscape, config = initialise()
    overrides = config["ImageSizeOverride"]
    multiplier = float(config["PixelMultiplier"])
    defaultwidth = int(config["DefaultImage"]["w"])
    defaultheight = int(config["DefaultImage"]["h"])

    i = 0
    for dir in dirs_images:
        svgs = []
        for file in os.listdir(dir):
            # Filter out non-svg files
            if file.endswith(".svg"):
                svgs.append("{p}/{f}".format(p=dir, f=file))
        svgs.sort()
        for svg in svgs:
            convertfile(inkscape, svg, defaultwidth, defaultheight,
                        multiplier, overrides, dirs_out[i])
        i += 1

    print("Done")
    exit(0)


if __name__ == '__main__':
    main()

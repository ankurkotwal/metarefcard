#!/usr/bin/env python3

import os
import re
import subprocess
from subprocess import STDOUT
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
    devices = {}
    with open("config/devices.yaml", "r") as stream:
        try:
            devices = yaml.safe_load(stream)
        except yaml.YAMLError as exc:
            print("Error loading devices.yaml {e}".format(e=exc))
    if DEBUG_OUTPUT:
        print("Devices loaded: {c}\n".format(c=config))
    config["ImageSizeOverride"] = devices["ImageSizeOverride"]

    # Find inkscape binary
    inkscape = subprocess.run(
        ["which", "inkscape"], capture_output=True).stdout.decode("utf-8").rstrip("\n")
    if False == os.path.isfile(inkscape):
        print("inkscape not found in path")
        exit(2)
    if DEBUG_OUTPUT:
        print("Found inkscape at {i}".format(i=inkscape))

    # Get inkscape version
    inkscapeVerCheck = subprocess.run(
        ["inkscape", "--version"], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL).stdout.decode("utf-8").rstrip("\n")
    version = re.search('^Inkscape\s+(\d+)\.(\d+)', inkscapeVerCheck)
    inkscapeVer = [version.group(1), version.group(2)]
    if inkscapeVer[0] != "1" and not(inkscapeVer[0] == "0" and inkscapeVer[1] == "92"):
        print("inkscape unknown version {v}".format(v=inkscapeVer))
        exit(2)
    if DEBUG_OUTPUT:
        print("Found inkscape version: {v}".format(v=inkscapeVer))

    # Find inkscape binary
    convert = subprocess.run(
        ["which", "convert"], capture_output=True).stdout.decode("utf-8").rstrip("\n")
    if False == os.path.isfile(inkscape):
        print("convert not found in path")
        exit(2)
    if DEBUG_OUTPUT:
        print("Found convert at {i}".format(i=inkscape))

    dir_hotas_images = "resources-source/hotas-images"
    checkDirExists(dir_hotas_images)
    # Confirm destination directory
    dir_hotas_out = config["HotasImagesDir"]
    ensureDirExists(dir_hotas_out)
    dir_logos = "resources-source/game-logos"
    checkDirExists(dir_logos)
    # Confirm destination directory
    dir_logos_out = config["LogoImagesDir"]
    ensureDirExists(dir_logos_out)

    return dir_hotas_images, dir_hotas_out, inkscape, inkscapeVer, dir_logos, dir_logos_out, convert, config


def checkDirExists(dir):
    # Confirm source images exist
    if False == os.path.isdir(dir):
        print("Error could not find dir {d}".format(d=dir))
        exit(1)
    if DEBUG_OUTPUT:
        print("Found hotas images at {d}".format(d=dir))


def ensureDirExists(dir):
    if False == os.path.isdir(dir):
        os.mkdir(dir)
        # Check that mkdir succeeded
        if False == os.path.isdir(dir):
            print("mkdir {d} failed".format(d=dir))
            exit(3)
    if DEBUG_OUTPUT:
        print("Found destination dir {d}".format(d=dir))


def convertfile(inkscape, inkscapeVer, svg, defaultwidth, defaultheight, multiplier,
                overrides, dir_out):
    name = os.path.splitext(os.path.basename(svg))[0]
    out = "{dir}/{out}.png".format(dir=dir_out, out=name)

    # Calculate new resolution
    width = int(defaultwidth * multiplier)
    height = int(defaultheight * multiplier)
    if name in overrides:
        width = int(overrides[name]["w"] * multiplier)
        height = int(overrides[name]["h"] * multiplier)

    # TODO Use Inkscape version
    # Convert svg to png with Inkscape
    cmd_export = [inkscape,
                  "--export-png={o}".format(o=out),
                  "-w={w}".format(w=width),
                  "-h={h}".format(h=height),
                  svg]
    if inkscapeVer[0] == "1":
        # Version 1 changed the command line
        cmd_export = [inkscape,
                      "-o", out,
                      "-w", "{w}".format(w=width),
                      "-h", "{h}".format(h=height),
                      svg]
    convert = subprocess.run(cmd_export,
                             stderr=subprocess.PIPE,
                             stdout=subprocess.PIPE)
    if convert.returncode != 0:
        print("Error: Failed to convert {f}".format(f=name))
    else:
        print("Converted {f}".format(f=name))


def resizefile(convert, img, height, multiplier, dir_out):
    name = os.path.splitext(os.path.basename(img))[0]
    out = "{dir}/{out}.png".format(dir=dir_out, out=name)

    # Convert svg to png with imagemagick
    cmd_export = [convert,
                  "-geometry",
                  "x{m}".format(m=multiplier * height),
                  img,
                  out]
    convert = subprocess.run(cmd_export,
                             stderr=subprocess.PIPE,
                             stdout=subprocess.PIPE)
    if convert.returncode != 0:
        print("Error: Failed to resize {f}".format(f=name))
    else:
        print("Resized {f}".format(f=name))


def main():
    dir_hotas_images, dir_hotas_out, inkscape, inkscapeVer, dir_logos, dir_logos_out, convert, config = initialise()
    overrides = config["ImageSizeOverride"]
    multiplier = float(config["PixelMultiplier"])
    defaultwidth = int(config["DefaultImage"]["w"])
    defaultheight = int(config["DefaultImage"]["h"])
    backgroundHeight = int(config["ImageHeader"]["BackgroundHeight"])

    logos = []
    for file in os.listdir(dir_logos):
        # Filter out irrelelvant files
        if file.endswith(".png"):
            logos.append("{p}/{f}".format(p=dir_logos, f=file))
    logos.sort()
    for logo in logos:
        resizefile(convert, logo, backgroundHeight, multiplier, dir_logos_out)

    hotases = []
    for file in os.listdir(dir_hotas_images):
        # Filter out irrelelvant files
        if file.endswith(".svg"):
            hotases.append("{p}/{f}".format(p=dir_hotas_images, f=file))
    hotases.sort()
    for hotas in hotases:
        convertfile(inkscape, inkscapeVer, hotas, defaultwidth, defaultheight,
                    multiplier, overrides, dir_hotas_out)

    print("Done")
    exit(0)


if __name__ == '__main__':
    main()

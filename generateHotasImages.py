#!/usr/bin/env python3

import sys, yaml, subprocess, os.path

dir_hotas_images = "assets/hotas_images"
# Confirm source images exist
if False == os.path.isdir(dir_hotas_images):
    print("Error could not find dir {d}".format(d=dir_hotas_images))
    exit(1)
print("Found hotas images at {d}".format(d=dir_hotas_images))

# Find inkscape binary
inkscape = subprocess.run(["which", "inkscape"], capture_output=True).stdout.decode("utf-8").rstrip("\n")
if False == os.path.isfile(inkscape):
    print("Inkscape not found in path")
    exit(2)
print("Found Inkscape at {i}".format(i=inkscape))

# Confirm destination directory
dir_out = "refcard/resources/hotas_images2"
if False == os.path.isdir(dir_out):
    os.mkdir(dir_out)
    # Check that mkdir succeeded
    if False == os.path.isdir(dir_out):
        print("mkdir {d} failed".format(d=dir_out))
        exit(3)
print("Found destination dir {d}".format(d=dir_out))


# Convert svg to png with Inkscape
cmd_inkscape = [inkscape, "--export-type=png", ""]

# config = {}
# with open("refcard/configs/config.yaml", "r") as stream:
#     try:
#         config = yaml.safe_load(stream)
#     except yaml.YAMLError as exc:
#         print(exc)

# print("Config:{c}\n".format(c=config))

# outDir = "refcard/" + config["ImagesDir"]
# print("== {s} ==".format(s=outDir))

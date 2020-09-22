#!/usr/bin/env python3

import sys, yaml, subprocess, os

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
dir_out = "refcard/resources/hotas_images"
if False == os.path.isdir(dir_out):
    os.mkdir(dir_out)
    # Check that mkdir succeeded
    if False == os.path.isdir(dir_out):
        print("mkdir {d} failed".format(d=dir_out))
        exit(3)
print("Found destination dir {d}".format(d=dir_out))

svgs = []
for file in os.listdir(dir_hotas_images):
    # Filter out non-svg files
    if file.endswith(".svg"):
        svgs.append("{p}/{f}".format(p=dir_hotas_images, f=file))

svgs.sort()
for file in svgs:
   out = os.path.basename(file).rstrip("svg") + "png"
   # Convert svg to png with Inkscape
   cmd_inkscape = [inkscape, "--export-type=png", file, "--export-filename={dir}/{out}".format(dir=dir_out,out=out)]
   convert = subprocess.run(cmd_inkscape, stderr=subprocess.DEVNULL, stdout=subprocess.DEVNULL)
   if convert.returncode != 0:
       print("Failed to convert {f}".format(f=out))
   else:
       print("Converted {f}".format(f=out))

print("Done")
exit(0)

# TODO - resize files

# config = {}
# with open("refcard/configs/config.yaml", "r") as stream:
#     try:
#         config = yaml.safe_load(stream)
#     except yaml.YAMLError as exc:
#         print(exc)

# print("Config:{c}\n".format(c=config))

# outDir = "refcard/" + config["ImagesDir"]
# print("== {s} ==".format(s=outDir))

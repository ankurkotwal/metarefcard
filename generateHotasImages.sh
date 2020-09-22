#!/bin/bash

BIN_INKSCAPE=`which inkscape`
# TODO Check if inkscape in path

BIN_CONVERT=`which convert`
# TODO Check if convert in path

HOTAS_IMAGES_DIR='assets/hotas_images'
OUT_DIR='refcard/resources/hotas_images'

for fullname in $HOTAS_IMAGES_DIR/*.svg
do
    filename=`basename "$fullname"`
    shortname=${filename%.*}
    CMD="$BIN_INKSCAPE --export-type=png \"$HOTAS_IMAGES_DIR/$shortname.svg\" | $BIN_CONVERT - \"$HOTAS_IMAGES_DIR/$shortname.jpg\""
    echo $CMD
    # `$CMD`
    exit $?
done


CMD="$BIN_INKSCAPE --export-type=png --export-filename=- \"$HOTAS_IMAGES_DIR/$shortname.svg\" | $BIN_CONVERT - \"$HOTAS_IMAGES_DIR/$shortname.jpg\""
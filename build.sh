#!/bin/bash

if [ `uname` == "Darwin" ]; then
    brew_path=`which brew`
    if [ $? -eq 0 ]; then
        jpeg_turbo_path=`brew --prefix jpeg-turbo 2>/dev/null`
        jpeg_turbo_version=`brew list --versions jpeg-turbo 2>/dev/null | sed -E 's/^jpeg-turbo[[:space:]]+//'`
        if [ $? -eq 0 ]; then
          jpeg_turbo_path="$jpeg_turbo_path/$jpeg_turb_version"
          export CGO_CFLAGS="-I${jpeg_turbo_path}include"
          export CGO_LDFLAGS="-L${jpeg_turbo_path}lib"
        fi
    fi
fi

go build

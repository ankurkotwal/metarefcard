#!/bin/bash

if [ `uname` == "Darwin" ]; then
    if [ -n "$HOMEBREW_PREFIX" ]; then
        export CGO_CFLAGS="-I ${HOMEBREW_PREFIX}/include"
        export CGO_LDFLAGS="-L ${HOMEBREW_PREFIX}/lib"
    fi
fi
go run -v ./... $@

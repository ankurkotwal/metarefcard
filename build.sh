#!/bin/bash

if [ `uname` == "Darwin" ] && [ -n "$HOMEBREW_PREFIX" ]; then
    export CGO_CFLAGS="-I ${HOMEBREW_PREFIX}/include"
    export CGO_LDFLAGS="-L ${HOMEBREW_PREFIX}/lib"
fi

go build -v ./...

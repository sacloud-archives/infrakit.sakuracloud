#!/bin/bash

set -e
#set -x

mkdir -p build/ 2>/dev/null

for GOOS in $OS; do
    for GOARCH in $ARCH; do
        arch="$GOOS-$GOARCH"
        binary="infrakit-instance-sakuracloud"
        if [ "$GOOS" = "windows" ]; then
          binary="${binary}.exe"
        fi
        echo "Building $binary $arch"
        GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 \
            go build \
                -ldflags "$BUILD_LDFLAGS" \
                -o build/$binary \
                plugin/instance/cmd/main.go
        if [ -n "$ARCHIVE" ]; then
            (cd build/; zip -r "infrakit-instance-sakuracloud_$arch" $binary)
            rm -f build/$binary
        fi
    done
done

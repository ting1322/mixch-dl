#!/usr/bin/bash

set -e

NDK=$HOME/sdk/ndk_standalone

CGO_ENABLED=1 \
    GOOS=android \
    GOARCH=arm64 \
    CC=$NDK/bin/aarch64-linux-android24-clang \
    CXX=$NDK/bin/aarch64-linux-android24-clang \
    go build

zip mixch-dl-android-arm64.zip mixch-dl
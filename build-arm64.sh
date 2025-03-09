#!/bin/bash

set -e

ver=`cat version.txt`-`git rev-parse --short HEAD`

rm -f mixch-dl mixch-dl.exe mixch-dl-linux-x86-64.zip mixch-dl-windows-x86-64.zip

set -x

GOOS=linux GOARCH=arm64 go build -o mixch-dl -ldflags "-X main.programVersion=$ver"


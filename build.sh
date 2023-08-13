#!/bin/bash

set -e

ver=`cat version.txt`-`git rev-parse --short HEAD`

rm -f mixch-dl mixch-dl.exe mixch-dl-linux-x86-64.zip mixch-dl-windows-x86-64.zip

set -x

go test github.com/ting1322/mixch-dl/mixch \
   github.com/ting1322/mixch-dl/m3u8 \
   github.com/ting1322/mixch-dl/twitcasting

go build -o mixch-dl -ldflags "-X main.programVersion=$ver"

zip mixch-dl-linux-x86-64.zip mixch-dl

GOOS=windows go build -o mixch-dl.exe -ldflags "-X main.programVersion=$ver"

zip mixch-dl-windows-x86-64.zip mixch-dl.exe
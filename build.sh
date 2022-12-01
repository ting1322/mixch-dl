#!/bin/bash

set -e

rm -f mixch-dl mixch-dl.exe mixch-dl-linux-x86-64.zip mixch-dl-windows-x86-64.zip

go test mixch m3u8 twitcasting

go build -o mixch-dl

zip mixch-dl-linux-x86-64.zip mixch-dl

GOOS=windows go build -o mixch-dl.exe

zip mixch-dl-windows-x86-64.zip mixch-dl.exe
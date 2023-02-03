#!/bin/bash

set -e

ver=`cat version.txt`-`git rev-parse --short HEAD`

set -x

go install -ldflags "-X main.programVersion=$ver"
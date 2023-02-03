#!/usr/bin/bash
#
# /home/ting/umi/start-mixch.sh
#

source ~/.profile

cd `dirname $0`

logfile=mixch-`date +%Y-%m-%d_%H-%M-%S`.log

mixch-dl --version > $logfile

set -x

mixch-dl -loop https://mixch.tv/u/17209506/live >> $logfile 2>&1

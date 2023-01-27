#!/usr/bin/bash
#
# /home/ting/umi/start-mixch.sh
#

cd `dirname $0`

logfile=mixch-`date +%Y-%m-%d_%H-%M-%S`.log

mixch-dl --version > $logfile

set -x

~/go/bin/mixch-dl -loop https://mixch.tv/u/17209506/live >> $logfile 2>&1
#!/usr/bin/bash
#
# /home/ting/umi/start-tc.sh
#

cd `dirname $0`

logfile=tc-`date +%Y-%m-%d_%H-%M-%S`.log

mixch-dl --version > $logfile

set -x

~/go/bin/mixch-dl -loop https://twitcasting.tv/0007umi >> $logfile 2>&1
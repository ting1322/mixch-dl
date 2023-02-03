#!/usr/bin/bash
#
# /home/ting/umi/start-tc.sh
#

source ~/.profile

cd `dirname $0`

logfile=tc-`date +%Y-%m-%d_%H-%M-%S`.log

mixch-dl --version > $logfile

set -x

mixch-dl -loop https://twitcasting.tv/0007umi >> $logfile 2>&1
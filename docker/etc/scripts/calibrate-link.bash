#!/bin/bash

set -eu

function mk-label() {
	local exid=$1
	local unixtime=$(date +%s)

	printf "lola-%s-%s" "${exid}" "${unixtime}"
}
REVERSE=${REVERSE:-FALSE}
TIME=${TIME:-30}

cwd=`dirname $0`
[ "$REVERSE" == "TRUE" ] && reverse="--reverse"
[ "$REVERSE" != "TRUE" ] && reverse=""

iperf3 -c iperf-server ${reverse} --time ${TIME} --logfile ${cwd}/$(mk-label "calibrate" ".txt")

#!/bin/bash

display_usage() {
    echo "Please supply function name to tail logs"
    echo "eg. ./slsLogs.sh opacheck"
}

if [ $# -eq 0 ]; then
    display_usage
    exit 1
fi

sls logs -v -t -f $1

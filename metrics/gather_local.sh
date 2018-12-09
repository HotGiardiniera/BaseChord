#!/bin/bash

# if [ $# -eq 0 ] || [ "$1" -lt 1 ]; then
#     echo "usage: gather_local.sh <num>"
#     exit 1
# fi

LOG_DIR=./local_logs/

if ! [[ -d "$LOG_DIR"  ]]; then
    echo "creating logs dir ..."
    mkdir $LOG_DIR
fi

echo "clearing old logs ..."
if [ -d "$LOG_DIR" ]; then rm -f $LOG_DIR/*; fi

cp /tmp/node*.JSON ./$LOG_DIR/

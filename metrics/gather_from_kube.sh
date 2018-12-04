#!/bin/bash

NUMPODS="$(kubectl get pods | wc -l)"
LOG_DIR=./logs/

if [ -d "$LOG_DIR" ]; then rm -f $LOG_DIR/*; fi

for i in `seq 0 "$(($NUMPODS - 2))"`
do
    node="chord$i"
    echo "getting $node metric file"
    if ! [[ -d "$LOG_DIR"  ]]; then
        echo "creating logs dir ..."
        mkdir "logs"
    fi
    kubectl cp default/$node:/tmp/ ./logs/ > /dev/null

done

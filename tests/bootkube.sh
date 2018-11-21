#!/bin/bash
# Copied from Aurojit Panda's lab2-raft code
set -E


if ! sudo minikube status > /dev/null; then 
    if [ $# -gt 0 ]; then
        sudo rm -rI $HOME/.kube/cache
    fi
    sudo minikube start --vm-driver=none
else
    echo 'Minikube is already running. Use `sudo minikube stop` to stop it if necessary'
fi
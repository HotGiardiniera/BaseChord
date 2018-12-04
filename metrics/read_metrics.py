#!/usr/bin/env python3
import os
import sys
import json
import subprocess

# FORMAT FOR PYTHON datetime.datetime.strptime(t1, "%Y-%m-%dT%H:%M:%S.%f")

CHORD_IMAGE = "local-chord-node"
LOGS_DIR = "./logs/"

def get_stats(file_data):
    print(file_data)


def main():
    # Gather metrics
    print("Gathering metrics...")
    subprocess.call("./gather_from_kube.sh", shell=True)

    for metric_file in os.listdir(LOGS_DIR):
        relative_path = LOGS_DIR+metric_file
        with open(relative_path) as f:
            print("Checking", relative_path)
            file_data = list()
            for line in f:
                data = json.loads(line)
                file_data.append(data)
            get_stats(file_data)


# Potential TODO draw chord map

if __name__ == "__main__":
    main()
    sys.exit(0)
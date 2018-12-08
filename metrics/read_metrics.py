#!/usr/bin/env python3
import os
import sys
import json
import subprocess
import re

# FORMAT FOR PYTHON datetime.datetime.strptime(t1, "%Y-%m-%dT%H:%M:%S.%f")

CHORD_IMAGE = "local-chord-node"
LOGS_DIR = "./logs/"

def print_ring_basic(node_list):
    node_list.sort()
    dumb_ring = "-->"
    for i in node_list:
        dumb_ring += (str(i)+"--")
    print(dumb_ring+">")

def get_stats(file_data):
    print(file_data)


def main():
    # Gather metrics
    print("Gathering metrics...")
    subprocess.call("./gather_from_kube.sh", shell=True)

    node_ids = list()

    for metric_file in os.listdir(LOGS_DIR):
        relative_path = LOGS_DIR+metric_file
        with open(relative_path) as f:
            print("Checking", relative_path)
            node_id = re.findall(r'\d+', relative_path)
            if node_id:
                node_ids.append(int(node_id[-1]))
            file_data = list()
            for line in f:
                data = json.loads(line)
                file_data.append(data)
            # get_stats(file_data)
    print_ring_basic(node_ids)


# Potential TODO draw chord map

if __name__ == "__main__":
    main()
    sys.exit(0)
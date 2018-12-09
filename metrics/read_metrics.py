#!/usr/bin/env python3
import os
import sys
import json
import subprocess
import re
from itertools import groupby

M = 7 # Size of finger table
REQUST_CLASS = 1
FINGER_CLASS = 2

# FORMAT FOR PYTHON datetime.datetime.strptime(t1, "%Y-%m-%dT%H:%M:%S.%f")

CHORD_IMAGE = "local-chord-node"
LOGS_DIR = "./logs/"

def print_ring_basic(node_list):
    node_list.sort()
    dumb_ring = "-->"
    for i in node_list:
        dumb_ring += (str(i)+"--")
    print(dumb_ring+">")

def get_stats(file_data, node_id):
    classes = dict()

    keyfunc = lambda x: x["Class"]

    file_data = sorted(file_data, key=keyfunc)
    for k, g in groupby(file_data, keyfunc):
        classes[k] = list(g)

    # Request hop metrics

    total_hops = 0
    misses = 0 # requests we needed to forward
    for request in classes[REQUST_CLASS]:
        # print(request)
        total_hops += request['Hops']
        misses += 1
    print("\n----------------------------")
    print("Node: ", node_id)
    print("Total Hops:", total_hops)
    print("Total Forwards:", misses)
    print("Hops per forward:", (total_hops/misses))
    print()


def main():
    # Gather metrics
    # print("Gathering metrics...")
    # subprocess.call("./gather_from_kube.sh", shell=True)

    node_ids = list()
    n = 0
    for metric_file in os.listdir(LOGS_DIR):
        relative_path = LOGS_DIR+metric_file
        with open(relative_path) as f:
            # print("Checking", relative_path)
            node_id = re.findall(r'\d+', relative_path)
            if node_id:
                node_ids.append(int(node_id[-1]))
            file_data = list()
            for line in f:
                data = json.loads(line)
                file_data.append(data)
            get_stats(file_data, node_ids[-1])
    # print_ring_basic(node_ids)



if __name__ == "__main__":
    main()
    sys.exit(0)
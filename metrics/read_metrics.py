#!/usr/bin/env python3
import os
import sys
import json
import subprocess
import re
from itertools import groupby
import csv
import datetime

M = 7 # Size of finger table
REQUST_CLASS = 1
FINGER_CLASS = 2

# FORMAT FOR PYTHON datetime.datetime.strptime(t1, "%Y-%m-%dT%H:%M:%S.%f")

CHORD_IMAGE = "local-chord-node"
LOGS_DIR = "./logs_no_add_16-50/"

def write_to_csv(info, node_list):
    with open('hops.csv'.format(), 'w') as f:
        for node in node_list:
            # Partition into hop
            for req in info[node][REQUST_CLASS]:
                start = datetime.datetime.strptime(req['Start'], "%Y-%m-%dT%H:%M:%S.%f")
                end = datetime.datetime.strptime(req['End'], "%Y-%m-%dT%H:%M:%S.%f")
                row = [req["Hops"], ((end-start).microseconds)/1000]
                writer = csv.writer(f)
                writer.writerow(row)
    with open('fingers.csv'.format(), 'w') as f:
        for node in node_list:
            # Partition into hop
            # print(info[node][FINGER_CLASS])
            for req in info[node][FINGER_CLASS]:
                row = [req["SourceNode"], req["FixedFingers"], req["Time"]]
                writer = csv.writer(f)
                writer.writerow(row)

def print_ring_basic(node_list):
    node_list.sort()
    dumb_ring = "-->"
    for i in node_list:
        dumb_ring += ("\u001b[36m"+str(i)+"\u001b[0m"+"--")
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
    time = 0
    start = 0
    end = 0
    num_files = -1
    if classes.get(REQUST_CLASS):
        for request in classes[REQUST_CLASS]:
            # print(request)
            if num_files < 0:
                num_files = request['NumFiles']
            total_hops += request['Hops']
            misses += 1
            start = datetime.datetime.strptime(request['Start'], "%Y-%m-%dT%H:%M:%S.%f")
            end = datetime.datetime.strptime(request['End'], "%Y-%m-%dT%H:%M:%S.%f")
            time += (end - start).microseconds
        # print("\n----------------------------")
        # print("Node: ", node_id)
        # print("Total Hops:", total_hops)
        # print("Total Time for Requests:", time)
        # print("Total Forwards:", misses)
        # print("Hops per forward:", (total_hops/misses))
        # print("Avg time per request:", (time/misses))
        # print("Files at node", num_files)
        # print()
    return classes


def main():
    # Gather metrics
    print("Gathering metrics...")
    subprocess.call("./gather_from_kube.sh", shell=True)

    node_ids = list()
    node_stats = dict()
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
            node_stats[int(node_id[-1])] = get_stats(file_data, node_ids[-1])
    print_ring_basic(node_ids)
    write_to_csv(node_stats, node_ids)
    # print(node_stats[82])
    # for k in node_ids:
    #     print("Node ", k)



if __name__ == "__main__":
    main()
    sys.exit(0)
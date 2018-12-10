#!/usr/bin/env python3
import os
import sys
import subprocess
import time

import kube

def main():
    # Launch chord ring with 8 nodes
    subprocess.call("./kube.py boot 8 0", shell=True)
    print("Launched ring")
    # Wait 15 seconds for ring to stabilize
    time.sleep(15)
    # Send 50 file requests
    print("Staring first 50...")
    subprocess.call("../client/client -kube -files=50", shell=True)
    print("Made first 50")

    # Add 4 more nodes to ring
    subprocess.call("./kube.py add 4 0", shell=True)
    print("Added 4 nodes")
    # Wait 20 seconds for ring to stabilize
    time.sleep(30)
    # Send 50 file requests
    subprocess.call("../client/client -kube -files=50", shell=True)
    print("Made second 50")

    # Add 4 more nodes to ring
    subprocess.call("./kube.py add 4 0", shell=True)
    print("Added 4 nodes")
    # Wait 20 seconds for ring to stabilize
    time.sleep(30)
    # Send 50 file requests
    subprocess.call("../client/client -kube -files=50", shell=True)
    print("Made third 50")


if __name__ == "__main__":
    main()
    sys.exit(0)

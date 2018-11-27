#!/bin/sh
set -E  # Inherit error trap
docker build --rm -t local-chord-node -f Dockerfile ../
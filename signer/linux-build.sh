#!/bin/bash
set -euo pipefail

# Build the whole cmd/stevie package (never a single file: a second file in
# package main would be silently excluded), stamped for traceability.
mkdir -p bin/linux
GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o bin/linux/stevie \
    ./cmd/stevie

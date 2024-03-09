#!/bin/bash
mkdir -p bin/linux
GOOS=linux GOARCH=amd64 go build -o bin/linux/stevie cmd/stevie/main.go

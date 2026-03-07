#!/bin/bash
set -e

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
go build -o lbs-server ./cmd/server
echo "Build completed successfully."
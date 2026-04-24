#!/usr/bin/env bash

set -euo pipefail

# Build cross-platform binaries and archive artifacts into dist/.
# This script is shared by:
# - GitHub Actions release workflow
# - Local/manual npm publish flow

rm -rf dist
mkdir -p dist

echo "Building cross-platform binaries..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/postx-linux-amd64 ./cmd/postx
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/postx-linux-arm64 ./cmd/postx
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/postx-darwin-amd64 ./cmd/postx
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/postx-darwin-arm64 ./cmd/postx
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/postx-windows-amd64.exe ./cmd/postx

echo "Creating release archives..."
tar -C dist -czf dist/postx-linux-amd64.tar.gz postx-linux-amd64
tar -C dist -czf dist/postx-linux-arm64.tar.gz postx-linux-arm64
tar -C dist -czf dist/postx-darwin-amd64.tar.gz postx-darwin-amd64
tar -C dist -czf dist/postx-darwin-arm64.tar.gz postx-darwin-arm64
zip -j dist/postx-windows-amd64.zip dist/postx-windows-amd64.exe

echo "Release assets ready in dist/"

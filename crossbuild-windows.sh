#!/bin/sh
# Cross-compile the Wails GUI for Windows from a Linux/macOS host.
# Requires: wails3 CLI (go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16),
# Node/npm or bun for the frontend build, and Go 1.26+.
GOOS=windows wails3 build

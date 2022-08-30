#!/bin/bash

BUILD_TIME=$(date -u '+%Y-%m-%d %I:%M:%S %Z')
BUILD_COMMIT=$(git rev-parse HEAD)

gox -osarch="darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64" \
        -output="dist/{{.Dir}}_{{.OS}}_{{.Arch}}" \
        -ldflags " \
          -X 'gogs.io/gogs/internal/conf.BuildTime=${BUILD_TIME}' \
          -X 'gogs.io/gogs/internal/conf.BuildCommit=${BUILD_COMMIT}'"

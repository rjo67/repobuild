#!/bin/bash

set -e

BINDIR=../bin

(
  cd go
  go build -o $BINDIR/repobuild.exe cmd/build/main.go
  go build -o $BINDIR/testserver.exe cmd/testserver/main.go
)



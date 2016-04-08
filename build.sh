#!/bin/bash

export GO15VENDOREXPERIMENT=1
go get github.com/tools/godep

godep restore
godep go build ./...
godep go test ./...

#!/bin/bash
mkdir -p gopath
mkdir -p vendor/src/github.com/arkenio/

export GO15VENDOREXPERIMENT=1

ln -sf `pwd` vendor/src/github.com/arkenio/goarken

go get github.com/mattn/gom
eval $(gom env | grep GOPATH)
gom install
gom build

#!/bin/bash
set -x
mkdir -p gopath
mkdir -p _vendor/src/github.com/arkenio/
ln -sf `pwd` _vendor/src/github.com/arkenio/goarken

go get github.com/mattn/gom
eval $(gom env | grep GOPATH)
gom install
gom build

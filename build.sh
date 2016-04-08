#!/bin/bash
set -x
mkdir -p gopath
mkdir -p _vendor/src/github.com/arkenio/
ln -sf `pwd` _vendor/src/github.com/arkenio/goarken

export GOPATH=`pwd`/gopath

go get github.com/mattn/gom
gom install
gom build

#!/bin/bash

export GO15VENDOREXPERIMENT=1

eval $(gom env | grep GOPATH)

$HOME/gopath/bin/gom test ./storage
$HOME/gopath/bin/gom test ./model
$HOME/gopath/bin/gom test ./drivers
$HOME/gopath/bin/gom test


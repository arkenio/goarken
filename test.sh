#!/bin/bash

export GOPATH=`pwd`/gopath

$HOME/gopath/bin/gom test ./storage
$HOME/gopath/bin/gom test ./model
$HOME/gopath/bin/gom test ./drivers
$HOME/gopath/bin/gom test


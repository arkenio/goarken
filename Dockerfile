FROM golang:1.5
MAINTAINER      Damien METZLER <dmetzler@nuxeo.com>

ENV GO15VENDOREXPERIMENT=1
RUN go get github.com/tools/godep



ENV APP_DIR $GOPATH/src/github.com/arkenio/goarken
ADD . $APP_DIR
RUN cd $APP_DIR &&\
    godep restore && \
    godep go build ./... && \
    godep go test && \
    godep go test ./storage && \
    godep go test ./drivers && \
    godep go test ./model




FROM golang:1.8.3-alpine3.6 as gobuild
WORKDIR /
ENV GOPATH="/"
RUN apk update && apk add pkgconfig build-base bash autoconf automake libtool gettext openrc git libzmq zeromq-dev
COPY . .
COPY . /src/github.com/me-box/goZestClient/
RUN go get github.com/pebbe/zmq4
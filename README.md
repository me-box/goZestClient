# goZestClient

[![Build Status](https://travis-ci.org/Toshbrown/goZestClient.svg?branch=master)](https://travis-ci.org/Toshbrown/goZestClient)

A Golang Lib for [REST over ZeroMQ](https://github.com/jptmoore/zest)

## Starting server to test against

```bash
$ docker run -p 5555:5555 -p 5556:5556 -d --name zest --rm jptmoore/zest:v0.1.1 /app/zest/server.exe --secret-key-file example-server-key --enable-logging
$ docker logs zest -f
```

## Client for testing

In the ./client directory there is a simple client for testing this library with the zest server.
Run `go run ./client.js` to run with the default options.

```bash
Usage of client.go:
  -enable-logging
    	output debug information
  -format string
    	text, json, binary to set the message content type (default "JSON")
  -method string
    	set the mode of operation (default "OBSERVE")
  -path string
    	Set the uri path for POST and GET (default "/kv/foo")
  -payload string
    	Set the uri path for POST and GET (default "{\"name\":\"dave\", \"age\":30}")
  -request-endpoint string
    	set the request/reply endpoint (default "tcp://127.0.0.1:5555")
  -router-endpoint string
    	set the router/dealer endpoint (default "tcp://127.0.0.1:5556")
  -server-key string
    	Set the curve server key (default "vl6wu0A@XP?}Or/&BR#LSxn>A+}L)p44/W[wXL3<")
  -token string
    	Set set access token
```

## Running unit tests

```
./test/test.sh
```

## Development of databox was supported by the following funding

```
EP/N028260/1, Databox: Privacy-Aware Infrastructure for Managing Personal Data

EP/N028260/2, Databox: Privacy-Aware Infrastructure for Managing Personal Data

EP/N014243/1, Future Everyday Interaction with the Autonomous Internet of Things

EP/M001636/1, Privacy-by-Design: Building Accountability into the Internet of Things (IoTDatabox)

EP/M02315X/1, From Human Data to Personal Experience

```
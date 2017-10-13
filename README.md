# goZestClient

A go Lib for [REST over ZeroMQ](https://github.com/jptmoore/zest)

This is work inprogress to build a lib for [Databox](https://github.com/me-box/databox)

## Starting server

```bash
$ docker run -p 5555:5555 -p 5556:5556 -d --name zest --rm jptmoore/zest /app/zest/server.exe --secret-key 'EKy(xjAnIfg6AT+OGd?nS1Mi5zZ&b*VXA@WxNLLE' --enable-logging
$ docker logs zest -f
```


## Testing
```bash

cd test
go run client.go

```

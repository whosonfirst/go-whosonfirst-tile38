CWD=$(shell pwd)
GOPATH := $(CWD)

build:	rmdeps deps fmt bin

prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep

rmdeps:
	if test -d src; then rm -rf src; fi 

deps:   self
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-crawl"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-geojson"
	@GOPATH=$(GOPATH) go get -u "github.com/garyburd/redigo/redis"

fmt:
	go fmt *.go

bin:	self
	@GOPATH=$(GOPATH) go build -o bin/wof-tile38-index cmd/wof-tile38-index.go

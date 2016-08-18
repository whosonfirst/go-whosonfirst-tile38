CWD=$(shell pwd)
GOPATH := $(CWD)

build:	rmdeps deps fmt bin

prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-tile38; then rm -rf src/github.com/whosonfirst/go-whosonfirst-tile38; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-tile38
	cp -r index src/github.com/whosonfirst/go-whosonfirst-tile38/index
	cp -r concordances src/github.com/whosonfirst/go-whosonfirst-tile38/concordances

rmdeps:
	if test -d src; then rm -rf src; fi 

deps:   self
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-crawl"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-geojson"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-placetypes"
	@GOPATH=$(GOPATH) go get -u "github.com/garyburd/redigo/redis"

fmt:
	go fmt cmd/*.go
	go fmt index/*.go

bin:	self
	@GOPATH=$(GOPATH) go build -o bin/wof-tile38-index cmd/wof-tile38-index.go
	@GOPATH=$(GOPATH) go build -o bin/wof-tile38-index-concordances cmd/wof-tile38-index-concordances.go

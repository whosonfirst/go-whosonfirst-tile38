CWD=$(shell pwd)
GOPATH := $(CWD)

build:	rmdeps deps fmt bin

prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-tile38; then rm -rf src/github.com/whosonfirst/go-whosonfirst-tile38; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-tile38
	cp -r client src/github.com/whosonfirst/go-whosonfirst-tile38/client
	cp -r flags src/github.com/whosonfirst/go-whosonfirst-tile38/flags
	cp -r index src/github.com/whosonfirst/go-whosonfirst-tile38/index
	cp -r util src/github.com/whosonfirst/go-whosonfirst-tile38/util
	cp -r whosonfirst src/github.com/whosonfirst/go-whosonfirst-tile38/whosonfirst
	cp tile38.go src/github.com/whosonfirst/go-whosonfirst-tile38/
	cp -r vendor/* src/

rmdeps:
	if test -d src; then rm -rf src; fi 

deps:
	@GOPATH=$(GOPATH) go get -u "github.com/facebookgo/grace/gracehttp"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-sanitize"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-bbox"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-geojson-v2"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-index"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-log"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-placetypes"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-spr"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-timer"
	@GOPATH=$(GOPATH) go get -u "github.com/whosonfirst/go-whosonfirst-uri"
	@GOPATH=$(GOPATH) go get -u "github.com/garyburd/redigo/redis"
	@GOPATH=$(GOPATH) go get -u "github.com/tidwall/gjson"
	@GOPATH=$(GOPATH) go get -u "github.com/tidwall/pretty"

vendor-deps: rmdeps deps
	if test ! -d vendor; then mkdir vendor; fi
	if test -d vendor/src; then rm -rf vendor/src; fi
	cp -r src/* vendor/
	find vendor -name '.git' -print -type d -exec rm -rf {} +
	rm -rf src

fmt:
	go fmt cmd/*.go
	go fmt client/*.go
	go fmt flags/*.go
	go fmt index/*.go
	go fmt tile38.go
	go fmt util/*.go
	go fmt whosonfirst/*.go

bin:	self
	@GOPATH=$(GOPATH) go build -o bin/wof-tile38-index cmd/wof-tile38-index.go
	@GOPATH=$(GOPATH) go build -o bin/wof-tile38-nearby cmd/wof-tile38-nearby.go
	@GOPATH=$(GOPATH) go build -o bin/wof-tile38-bboxd cmd/wof-tile38-bboxd.go

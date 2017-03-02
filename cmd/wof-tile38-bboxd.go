package main

/*
	This assumes:

	* That this is a terrible name for an application - it will be renamed

	* Data that has been indexed by github:whosonfirst/go-whosonfirst-tile38/cmd/wof-tile38-index.go

	To do:

	* Handle cursors/pagination (from Tile38)
	* Use RESP protocol instead of HTTP (https://github.com/tidwall/tile38/wiki/Go-example-(redigo))

	For example:
	* curl 'localhost:8080?bbox=-33.893217,151.165524,-33.840479,151.281223'

*/

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/whosonfirst/go-whosonfirst-bbox/parser"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"github.com/whosonfirst/go-whosonfirst-tile38/whosonfirst"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {

	var host = flag.String("host", "localhost", "The address your HTTP server should listen for requests on")
	var port = flag.Int("port", 8080, "The port number your HTTP server should listen for requests on")

	var t38_host = flag.String("tile38-host", "localhost", "The address your Tile38 server is bound to.")
	var t38_port = flag.Int("tile38-port", 9851, "The port number your Tile38 server is bound to.")
	var t38_collection = flag.String("tile38-collection", "", "The name of the Tile38 collection to read data from.")

	flag.Parse()

	t38_addr := fmt.Sprintf("%s:%d", *t38_host, *t38_port)

	handler := func(rsp http.ResponseWriter, req *http.Request) {

		query := req.URL.Query()

		bbox := query.Get("bbox")
		scheme := query.Get("scheme")
		order := query.Get("order")

		// cursor := query.Get("cursor")

		if bbox == "" {
			http.Error(rsp, "Missing bbox parameter", http.StatusBadRequest)
			return
		}

		p, err := parser.NewParser()

		if err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		if scheme != "" {
			p.Scheme = scheme
		}

		if order != "" {
			p.Order = order
		}

		bb, err := p.Parse(bbox)

		if err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		swlat := bb.MinY()
		swlon := bb.MinX()
		nelat := bb.MaxY()
		nelon := bb.MaxX()

		// TO DO: CURSORS

		t38_cmd := fmt.Sprintf("INTERSECTS %s POINTS BOUNDS %0.6f %0.6f %0.6f %0.6f", *t38_collection, swlat, swlon, nelat, nelon)
		t38_url := fmt.Sprintf("http://%s/%s", t38_addr, url.QueryEscape(t38_cmd))

		log.Println(t38_url)

		http_rsp, err := http.Get(t38_url)

		if err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		defer http_rsp.Body.Close()

		results, err := ioutil.ReadAll(http_rsp.Body)

		if err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		var t38_rsp tile38.Tile38Response
		err = json.Unmarshal(results, &t38_rsp)

		if err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		wof_rsp, err := whosonfirst.Tile38ResponseToWOFResponse(t38_rsp)

		if err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		json_rsp, err := json.Marshal(wof_rsp)

		if err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		rsp.Header().Set("Access-Control-Allow-Origin", "*")
		rsp.Header().Set("Content-Type", "application/json")

		rsp.Write(json_rsp)
	}

	endpoint := fmt.Sprintf("%s:%d", *host, *port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	err := gracehttp.Serve(&http.Server{Addr: endpoint, Handler: mux})

	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

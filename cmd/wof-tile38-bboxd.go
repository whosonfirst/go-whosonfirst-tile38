package main

/*
	This assumes:

	* That this is a terrible name for an application - it will be renamed

	* Data that has been indexed by github:whosonfirst/go-whosonfirst-tile38/cmd/wof-tile38-index.go

	To do:
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
	"github.com/whosonfirst/go-whosonfirst-tile38/client"
	"github.com/whosonfirst/go-whosonfirst-tile38/whosonfirst"
	"log"
	"net/http"
	"os"
	_ "strings"
)

func main() {

	var host = flag.String("host", "localhost", "The address your HTTP server should listen for requests on")
	var port = flag.Int("port", 8080, "The port number your HTTP server should listen for requests on")

	var t38_host = flag.String("tile38-host", "localhost", "The address your Tile38 server is bound to.")
	var t38_port = flag.Int("tile38-port", 9851, "The port number your Tile38 server is bound to.")
	var t38_collection = flag.String("tile38-collection", "", "The name of the Tile38 collection to read data from.")

	flag.Parse()

	// t38_client, err := client.NewRESPClient(*t38_host, *t38_port)
	t38_client, err := client.NewHTTPClient(*t38_host, *t38_port)

	if err != nil {
		log.Fatal(err)
	}

	handler := func(rsp http.ResponseWriter, req *http.Request) {

		query := req.URL.Query()

		bbox := query.Get("bbox")
		scheme := query.Get("scheme")
		order := query.Get("order")

		cursor := query.Get("cursor")
		per_page := query.Get("per_page")

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

		t38_cmd := "INTERSECTS"

		// See this... Yeah, Go is weird that way...

		t38_args := []interface{}{
			*t38_collection,
		}

		if cursor != "" {
			t38_args = append(t38_args, "CURSOR")
			t38_args = append(t38_args, cursor)
		}

		if per_page != "" {

			t38_args = append(t38_args, "LIMIT")
			t38_args = append(t38_args, per_page)
		}

		t38_args = append(t38_args, "POINTS")
		t38_args = append(t38_args, "BOUNDS")
		t38_args = append(t38_args, fmt.Sprintf("%0.6f", swlat))
		t38_args = append(t38_args, fmt.Sprintf("%0.6f", swlon))
		t38_args = append(t38_args, fmt.Sprintf("%0.6f", nelat))
		t38_args = append(t38_args, fmt.Sprintf("%0.6f", nelon))

		t38_rsp, err := t38_client.Do(t38_cmd, t38_args...)

		if err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)
			return
		}

		wof_rsp, err := whosonfirst.Tile38ResponseToWOFResponse(t38_rsp.(tile38.Tile38Response))

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

	err = gracehttp.Serve(&http.Server{Addr: endpoint, Handler: mux})

	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

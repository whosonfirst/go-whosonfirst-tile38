package main

/*
	This assumes:

	* That this is a terrible name for an application - it will be renamed

	* Data that has been indexed by github:whosonfirst/go-whosonfirst-tile38/cmd/wof-tile38-index.go

	* Example: curl 'localhost:8080?bbox=-33.893217,151.165524,-33.840479,151.281223'

*/

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/whosonfirst/go-sanitize"
	"github.com/whosonfirst/go-whosonfirst-bbox/parser"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"github.com/whosonfirst/go-whosonfirst-tile38/client"
	"github.com/whosonfirst/go-whosonfirst-tile38/util"
	"github.com/whosonfirst/go-whosonfirst-tile38/whosonfirst"
	"log"
	"net/http"
	"os"
)

func main() {

	var host = flag.String("host", "localhost", "The address your HTTP server should listen for requests on")
	var port = flag.Int("port", 8080, "The port number your HTTP server should listen for requests on")

	var t38_host = flag.String("tile38-host", "localhost", "The address your Tile38 server is bound to.")
	var t38_port = flag.Int("tile38-port", 9851, "The port number your Tile38 server is bound to.")
	var t38_collection = flag.String("tile38-collection", "", "The name of the Tile38 collection to read data from.")

	flag.Parse()

	t38_client, err := client.NewRESPClient(*t38_host, *t38_port)

	if err != nil {
		log.Fatal(err)
	}

	handler := func(rsp http.ResponseWriter, req *http.Request) {

		query := req.URL.Query()

		opts := sanitize.DefaultOptions()

		bbox, err := sanitize.SanitizeString(query.Get("bbox"), opts)

		if err != nil {
			http.Error(rsp, "Invalid bbox parameter", http.StatusBadRequest)
			return
		}

		scheme, err := sanitize.SanitizeString(query.Get("scheme"), opts)

		if err != nil {
			http.Error(rsp, "Invalid scheme parameter", http.StatusBadRequest)
			return
		}

		order, err := sanitize.SanitizeString(query.Get("order"), opts)

		if err != nil {
			http.Error(rsp, "Invalid orderx parameter", http.StatusBadRequest)
			return
		}

		cursor, err := sanitize.SanitizeString(query.Get("cursor"), opts)

		if err != nil {
			http.Error(rsp, "Invalid cursor parameter", http.StatusBadRequest)
			return
		}

		per_page, err := sanitize.SanitizeString(query.Get("per_page"), opts)

		if err != nil {
			http.Error(rsp, "Invalid per_page parameter", http.StatusBadRequest)
			return
		}

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

		cmd := []string{
			fmt.Sprintf("INTERSECTS %s", *t38_collection),
		}

		if cursor != "" {
			cmd = append(cmd, fmt.Sprintf("CURSOR %s", cursor))
		}

		if per_page != "" {
			cmd = append(cmd, fmt.Sprintf("LIMIT %s", per_page))
		}

		cmd = append(cmd, fmt.Sprintf("POINTS BOUNDS %0.6f %0.6f %0.6f %0.6f", swlat, swlon, nelat, nelon))

		t38_cmd, t38_args := util.ListToRESPCommand(cmd)
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

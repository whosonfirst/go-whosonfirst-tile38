package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tidwall/pretty"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"github.com/whosonfirst/go-whosonfirst-tile38/client"
	"github.com/whosonfirst/go-whosonfirst-tile38/util"
	"github.com/whosonfirst/go-whosonfirst-tile38/whosonfirst"
	"log"
	"os"
)

func main() {

	var t38_host = flag.String("tile38-host", "localhost", "The address your Tile38 server is bound to.")
	var t38_port = flag.Int("tile38-port", 9851, "The port number your Tile38 server is bound to.")
	var t38_collection = flag.String("tile38-collection", "", "The name of the Tile38 collection to read data from.")

	var lat = flag.Float64("latitude", 0.0, "A valid latitude.")
	var lon = flag.Float64("longitude", 0.0, "A valid longitude.")
	var radius = flag.Int("radius", 20, "A valid radius (in meters).")

	var debug = flag.Bool("debug", false, "Print debugging information.")

	flag.Parse()

	t38_client, err := client.NewRESPClient(*t38_host, *t38_port)

	if err != nil {
		log.Fatal(err)
	}

	nearby_cmd := "NEARBY"

	nearby_args := []interface{}{
		*t38_collection,
		"POINTS", "POINT",
		fmt.Sprintf("%0.6f", *lat),
		fmt.Sprintf("%0.6f", *lon),
		fmt.Sprintf("%d", *radius),
	}

	if *debug {
		log.Println(util.RESPCommandToString(nearby_cmd, nearby_args))
	}

	t38_rsp, err := t38_client.Do(nearby_cmd, nearby_args...)

	if err != nil {
		log.Fatal(err)
	}

	err = util.EnsureOk(t38_rsp)

	if err != nil {
		log.Fatal(err)
	}

	wof_rsp, err := whosonfirst.Tile38ResponseToWOFResponse(t38_rsp.(tile38.Tile38Response))

	if err != nil {
		log.Fatal(err)
	}

	// sudo put this in a function somewhere...
	// sudo make me run concurrently

	for i, wof_row := range wof_rsp.Results {

		meta_cmd := "GET"

		meta_args := []interface{}{
			*t38_collection,
			fmt.Sprintf("%d#meta", wof_row.WOFID),
		}

		if *debug {
			log.Println(util.RESPCommandToString(meta_cmd, meta_args))
		}

		meta_rsp, err := t38_client.DoMeta(meta_cmd, meta_args...)

		if err != nil {
			log.Fatal(err)
		}

		if !meta_rsp.(tile38.Tile38MetaResponse).Ok {
			continue
		}

		wof_meta, err := whosonfirst.Tile38MetaResponseToWOFMetaResult(meta_rsp.(tile38.Tile38MetaResponse))

		if err != nil {
			log.Fatal(err)
		}

		wof_rsp.Results[i].WOFName = wof_meta.WOFName
		wof_rsp.Results[i].WOFCountry = wof_meta.WOFCountry

	}

	json_rsp, err := json.Marshal(wof_rsp)

	if err != nil {
		log.Fatal(err)
	}

	pretty_rsp := pretty.Pretty(json_rsp)

	fmt.Fprintf(os.Stdout, "%s", pretty_rsp)
	os.Exit(0)
}

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
	"strings"
)

func main() {

	var t38_host = flag.String("tile38-host", "localhost", "The address your Tile38 server is bound to.")
	var t38_port = flag.Int("tile38-port", 9851, "The port number your Tile38 server is bound to.")
	var t38_collection = flag.String("tile38-collection", "", "The name of the Tile38 collection to read data from.")

	var lat = flag.Float64("latitude", 0.0, "A valid latitude.")
	var lon = flag.Float64("longitude", 0.0, "A valid longitude.")
	var radius = flag.Int("radius", 20, "A valid radius (in meters).")

	flag.Parse()

	t38_client, err := client.NewRESPClient(*t38_host, *t38_port)

	if err != nil {
		log.Fatal(err)
	}

	cmd := []string{
		fmt.Sprintf("NEARBY %s", *t38_collection),
	}

	/*
		if cursor != "" {
			cmd = append(cmd, fmt.Sprintf("CURSOR %s", cursor))
		}

		if per_page != "" {
			cmd = append(cmd, fmt.Sprintf("LIMIT %s", per_page))
		}
	*/

	cmd = append(cmd, fmt.Sprintf("POINTS POINT %0.6f %0.6f %d", *lat, *lon, *radius))

	log.Println(strings.Join(cmd, " "))

	t38_cmd, t38_args := util.ListToRESPCommand(cmd)
	t38_rsp, err := t38_client.Do(t38_cmd, t38_args...)

	if err != nil {
		log.Fatal(err)
	}

	wof_rsp, err := whosonfirst.Tile38ResponseToWOFResponse(t38_rsp.(tile38.Tile38Response))

	if err != nil {
		log.Fatal(err)
	}

	json_rsp, err := json.Marshal(wof_rsp)

	if err != nil {
		log.Fatal(err)
	}

	pretty_rsp := pretty.Pretty(json_rsp)

	fmt.Fprintf(os.Stdout, "%s", pretty_rsp)

	os.Exit(0)
}

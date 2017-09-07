package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/feature"
	wof "github.com/whosonfirst/go-whosonfirst-index"
	"github.com/whosonfirst/go-whosonfirst-index/utils"
	"github.com/whosonfirst/go-whosonfirst-log"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"github.com/whosonfirst/go-whosonfirst-tile38/client"
	"github.com/whosonfirst/go-whosonfirst-tile38/index"
	"github.com/whosonfirst/go-whosonfirst-timer"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type Endpoints []string

func (e *Endpoints) String() string {
	return strings.Join(*e, "\n")
}

func (e *Endpoints) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func (e *Endpoints) ToClients() ([]tile38.Tile38Client, error) {

	clients := make([]tile38.Tile38Client, 0)

	for _, str_pair := range *e {

		pair := strings.Split(str_pair, ":")

		if len(pair) > 2 {
			msg := fmt.Sprintf("Invalid endpoint string %s", str_pair)
			return nil, errors.New(msg)
		}

		var host string
		var port int

		if len(pair) == 1 {
			host = pair[0]
			port = 9851
		} else {

			p, err := strconv.Atoi(pair[1])

			if err != nil {
				return nil, err
			}

			host = pair[0]
			port = p
		}

		t38_client, err := client.NewRESPClient(host, port)

		if err != nil {
			return nil, err
		}

		clients = append(clients, t38_client)
	}

	return clients, nil
}

func main() {

	var endpoints Endpoints

	flag.Var(&endpoints, "tile38-endpoint", "One or more Tile38 'host:port' (or simply 'host' in which case port is assumed to be '9851') endpoints to connect to.")

	valid_modes := strings.Join(wof.Modes(), ", ")

	mode := flag.String("mode", "files", "The mode to use importing data. Valid options are: "+valid_modes)
	geom := flag.String("geometry", "", "Which geometry to index. Valid options are: centroid, bbox or whatever is in the default GeoJSON geometry (default).")

	procs := flag.Int("procs", runtime.NumCPU()*2, "The number of concurrent processes to use importing data.")

	t38_host := flag.String("tile38-host", "localhost", "The address your Tile38 server is bound to. This flag has been deprecated and you should use -tile38-endpoint instead.")
	t38_port := flag.Int("tile38-port", 9851, "The port number your Tile38 server is bound to. This flag has been deprecated and you should use -tile38-endpoint instead.")

	t38_collection := flag.String("tile38-collection", "", "The name of the Tile38 collection for indexing data.")

	lax := flag.Bool("lax", false, "Disable default strict checking when indexing files.")

	verbose := flag.Bool("verbose", false, "Be chatty about what's happening. This is automatically enabled if the -debug flag is set.")
	debug := flag.Bool("debug", false, "Go through all the motions but don't actually index anything.")

	flag.Parse()

	if *debug {
		*verbose = true
	}

	logger := log.SimpleWOFLogger()

	runtime.GOMAXPROCS(*procs)

	clients, err := endpoints.ToClients()

	if err != nil {
		logger.Fatal("failed to convert endpoints to clients because %v", err)
	}

	if len(clients) == 0 {

		t38_client, err := client.NewRESPClient(*t38_host, *t38_port)

		if err != nil {
			logger.Fatal("failed to create Tile38Client (%s:%d) because %v", *t38_host, *t38_port, err)
		}

		clients = append(clients, t38_client)
	}

	indexer, err := index.NewTile38Indexer(clients...)

	indexer.Verbose = *verbose
	indexer.Debug = *debug
	indexer.Geometry = *geom

	if *lax {
		indexer.Strict = false
	}

	cb := func(fh io.Reader, ctx context.Context, args ...interface{}) error {

		ok, err := utils.IsPrincipalWOFRecord(fh, ctx)

		if err != nil {
			return err
		}

		if !ok {
			return nil
		}

		f, err := feature.LoadWOFFeatureFromReader(fh)

		if err != nil {
			return err
		}

		return indexer.IndexFeature(f, *t38_collection)
	}

	wof_indexer, err := wof.NewIndexer(*mode, cb)

	if err != nil {
		logger.Fatal("Failed to create new indexer because %s", err)
	}

	tm, err := timer.NewDefaultTimer()

	if err != nil {
		logger.Fatal("Failed to create timer because %s", err)
	}

	defer tm.Stop()

	err = wof_indexer.IndexPaths(flag.Args())

	if err != nil {
		logger.Fatal("Failed to index paths in %s mode because %s", *mode, err)
	}

	os.Exit(0)
}

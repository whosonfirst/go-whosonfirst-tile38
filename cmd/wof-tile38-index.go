package main

import (
	"context"
	"flag"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/feature"
	wof "github.com/whosonfirst/go-whosonfirst-index"
	"github.com/whosonfirst/go-whosonfirst-index/utils"
	"github.com/whosonfirst/go-whosonfirst-log"
	"github.com/whosonfirst/go-whosonfirst-tile38/client"
	"github.com/whosonfirst/go-whosonfirst-tile38/index"
	"github.com/whosonfirst/go-whosonfirst-timer"
	"io"
	"os"
	"runtime"
)

func main() {

	mode := flag.String("mode", "files", "The mode to use importing data. Valid options are: directory, filelist and files.")
	geom := flag.String("geometry", "", "Which geometry to index. Valid options are: centroid, bbox or whatever is in the default GeoJSON geometry (default).")

	procs := flag.Int("procs", runtime.NumCPU()*2, "The number of concurrent processes to use importing data.")

	t38_host := flag.String("tile38-host", "localhost", "The address your Tile38 server is bound to.")
	t38_port := flag.Int("tile38-port", 9851, "The port number your Tile38 server is bound to.")
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

	t38_client, err := client.NewRESPClient(*t38_host, *t38_port)

	if err != nil {
		logger.Fatal("failed to create Tile38Client (%s:%d) because %v", *t38_host, *t38_port, err)
	}

	indexer, err := index.NewTile38Indexer(t38_client)

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

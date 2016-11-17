package main

import (
	"flag"
	"github.com/whosonfirst/go-whosonfirst-tile38/index"
	// "github.com/whosonfirst/go-whosonfirst-tile38/simple"
	"log"
	"os"
	"runtime"
	"strings"
)

func main() {

	mode := flag.String("mode", "files", "The mode to use importing data. Valid options are: directory, filelist and files.")
	geom := flag.String("geometry", "", "Which geometry to index. Valid options are: centroid, bbox or whatever is in the default GeoJSON geometry (default).")

	procs := flag.Int("procs", 200, "The number of concurrent processes to use importing data.")
	collection := flag.String("collection", "", "The name of the Tile38 collection for indexing data.")
	nfs_kludge := flag.Bool("nfs-kludge", false, "Enable the (walk.go) NFS kludge to ignore 'readdirent: errno' 523 errors")

	tile38_host := flag.String("tile38-host", "localhost", "The host of your Tile-38 server.")
	tile38_port := flag.Int("tile38-port", 9851, "The port of your Tile38 server.")

	verbose := flag.Bool("verbose", false, "Be chatty about what's happening. This is automatically enabled if the -debug flag is set.")
	debug := flag.Bool("debug", false, "Go through all the motions but don't actually index anything.")

	flag.Parse()

	if *debug {
		*verbose = true
	}

	runtime.GOMAXPROCS(*procs)

	client, err := tile38.NewTile38Client(*tile38_host, *tile38_port)

	if err != nil {
		log.Fatalf("failed to create Tile38Client (%s:%d) because %v", *tile38_host, *tile38_port, err)
	}

	client.Verbose = *verbose
	client.Debug = *debug
	client.Geometry = *geom

	args := flag.Args()

	for _, path := range args {

		if *mode == "directory" {

			err = client.IndexDirectory(path, *collection, *nfs_kludge)

		} else if *mode == "filelist" {

			err = client.IndexFileList(path, *collection)

		} else if *mode == "meta" {

			parts := strings.Split(path, ":")

			if len(parts) != 2 {
				log.Fatal("Invalid path declaration for a meta file")
			}

			for _, p := range parts {

				_, err := os.Stat(p)

				if os.IsNotExist(err) {
					log.Fatal("Path does not exist", p)
				}
			}

			meta_file := parts[0]
			data_root := parts[1]

			err = client.IndexMetaFile(meta_file, *collection, data_root)

		} else {
			err = client.IndexFile(path, *collection)
		}

		if err != nil {
			log.Fatalf("failed to index '%s' in (%s) mode, because %v", path, *mode, err)
		}
	}

	os.Exit(0)
}

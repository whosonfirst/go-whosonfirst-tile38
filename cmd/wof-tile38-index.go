package main

import (
	"flag"
	"github.com/whosonfirst/go-whosonfirst-tile38/client"
	"github.com/whosonfirst/go-whosonfirst-tile38/index"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {

	mode := flag.String("mode", "files", "The mode to use importing data. Valid options are: directory, filelist and files.")
	geom := flag.String("geometry", "", "Which geometry to index. Valid options are: centroid, bbox or whatever is in the default GeoJSON geometry (default).")

	procs := flag.Int("procs", runtime.NumCPU()*2, "The number of concurrent processes to use importing data.")
	nfs_kludge := flag.Bool("nfs-kludge", false, "Enable the (walk.go) NFS kludge to ignore 'readdirent: errno' 523 errors")

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

	runtime.GOMAXPROCS(*procs)

	t38_client, err := client.NewRESPClient(*t38_host, *t38_port)

	if err != nil {
		log.Fatalf("failed to create Tile38Client (%s:%d) because %v", *t38_host, *t38_port, err)
	}

	indexer, err := index.NewTile38Indexer(t38_client)

	indexer.Verbose = *verbose
	indexer.Debug = *debug
	indexer.Geometry = *geom

	if *lax {
		indexer.Strict = false
	}

	args := flag.Args()

	for _, path := range args {

		if *mode == "directory" {

			abs_path, err := filepath.Abs(path)

			if err != nil {
				log.Fatal(err)
			}

			err = indexer.IndexDirectory(abs_path, *t38_collection, *nfs_kludge)

		} else if *mode == "repo" {

			abs_path, err := filepath.Abs(path)

			if err != nil {
				log.Fatal(err)
			}

			data := filepath.Join(abs_path, "data")

			_, err = os.Stat(data)

			if err != nil {
				log.Fatal(err)
			}

			err = indexer.IndexDirectory(data, *t38_collection, *nfs_kludge)

		} else if *mode == "filelist" {

			abs_path, err := filepath.Abs(path)

			if err != nil {
				log.Fatal(err)
			}

			err = indexer.IndexFileList(abs_path, *t38_collection)

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

			err = indexer.IndexMetaFile(meta_file, *t38_collection, data_root)

		} else {

			abs_path, err := filepath.Abs(path)

			if err != nil {
				log.Fatal(err)
			}

			err = indexer.IndexFile(abs_path, *t38_collection)
		}

		if err != nil {

			log.Printf("failed to index '%s' in (%s) mode, because %v\n", path, *mode, err)

			if ! *lax {
				log.Fatal("Giving up")
			}
		}
	}

	os.Exit(0)
}

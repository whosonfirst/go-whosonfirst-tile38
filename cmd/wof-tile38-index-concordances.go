package main

	/*
	If this looks an awful lot like wof-tile38-index.go that's because
	it is. Please see notes at the top to concordances/concordances.go
	(20160818/thisisaaronland)
	*/

import (
	"flag"
	"github.com/whosonfirst/go-whosonfirst-tile38/concordances"
	"log"
	"os"
	"runtime"
)

func main() {

	mode := flag.String("mode", "files", "...")

	procs := flag.Int("procs", 200, "...")
	collection := flag.String("collection", "", "...")
	nfs_kludge := flag.Bool("nfs-kludge", false, "Enable the (walk.go) NFS kludge to ignore 'readdirent: errno' 523 errors")

	debug := flag.Bool("debug", false, "...")

	tile38_host := flag.String("tile38-host", "localhost", "...")
	tile38_port := flag.Int("tile38-port", 9851, "...")

	flag.Parse()

	runtime.GOMAXPROCS(*procs)

	client, err := tile38.NewTile38Client(*tile38_host, *tile38_port)

	if err != nil {
		panic(err)
	}

	client.Debug = *debug

	args := flag.Args()

	for _, path := range args {

		if *mode == "directory" {

			err = client.IndexDirectory(path, *collection, *nfs_kludge)

		} else if *mode == "filelist" {

			err = client.IndexFileList(path, *collection)

		} else {

			err = client.IndexFile(path, *collection)
		}

		if err != nil {
			log.Fatalf("failed to index %s in %s mode, because %v", path, *mode, err)
		}
	}

	os.Exit(0)
}

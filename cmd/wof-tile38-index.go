package main

import (
	"flag"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-tile38/index"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

func main() {

	root := flag.String("root", "", "...")
	procs := flag.Int("procs", 200, "...")
	collection := flag.String("collection", "", "...")
	nfs_kludge := flag.Bool("nfs-kludge", false, "Enable the (walk.go) NFS kludge to ignore 'readdirent: errno' 523 errors")

	tile38_host := flag.String("tile38-host", "localhost", "...")
	tile38_port := flag.Int("tile38-port", 9851, "...")

	flag.Parse()

	runtime.GOMAXPROCS(*procs)

	client, err := tile38.NewTile38Client(*tile38_host, *tile38_port)

	if err != nil {
		panic(err)
	}

	re_wof, _ := regexp.Compile(`(\d+)\.geojson$`)

	cb := func(abs_path string, info os.FileInfo) error {

		// please make me more like this...
		// https://github.com/whosonfirst/py-mapzen-whosonfirst-utils/blob/master/mapzen/whosonfirst/utils/__init__.py#L265

		fname := filepath.Base(abs_path)

		if !re_wof.MatchString(fname) {
			// log.Println("skip", abs_path)
			return nil
		}

		err := client.IndexFile(abs_path, *collection)

		if err != nil {
		       log.Printf("failed to index %s, because %v", abs_path, err)
		       return err
		}

		return nil
	}

	c := crawl.NewCrawler(*root)
	c.NFSKludge = *nfs_kludge

	_ = c.Crawl(cb)

}

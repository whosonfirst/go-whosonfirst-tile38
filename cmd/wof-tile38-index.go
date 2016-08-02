package main

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-geojson"
	"os"
	"runtime"
	"strconv"
)

func main() {

	root := flag.String("root", "", "...")
	procs := flag.Int("procs", 200, "...")
	nfs_kludge := flag.Bool("nfs-kludge", false, "Enable the (walk.go) NFS kludge to ignore 'readdirent: errno' 523 errors")

	// verbose := flag.Bool("verbose", false, "...")

	tile38_host := flag.String("tile38-host", "localhost", "...")
	tile38_port := flag.Int("tile38-port", 9851, "...")

	flag.Parse()

	runtime.GOMAXPROCS(*procs)

	tile38_endpoint := fmt.Sprintf("%s:%d", *tile38_host, *tile38_port)
	fmt.Println("connect to", tile38_endpoint)

	cb := func(abs_path string, info os.FileInfo) error {

		// please put me in a package specific function
		// fmt.Println("index", abs_path)

		feature, err := geojson.UnmarshalFile(abs_path)

		if err != nil {
			return err
		}

		wofid := feature.Id()
		str_wofid := strconv.Itoa(wofid)

		if err != nil {
			return err
		}

		body := feature.Body()
		geom := body.Path("geometry")

		str_geom := geom.String()

		conn, err := redis.Dial("tcp", tile38_endpoint)

		if err != nil {
			return nil
		}

		defer conn.Close()

		_, err = conn.Do("SET", "whosonfirst", str_wofid, "OBJECT", str_geom)

		if err != nil {
			return err
		}

		// http://tile38.com/commands/set/
		// http://tile38.com/commands/fset/

		placetype := feature.Placetype()
		key := str_wofid + ":placetype"

		_, err = conn.Do("SET", "whosonfirst", key, "STRING", placetype)

		if err != nil {
			fmt.Printf("FAILED to set placetype on %s because, %v\n", key, err)
		}

		// please set hierarchy information

		return nil
	}

	c := crawl.NewCrawler(*root)
	c.NFSKludge = *nfs_kludge

	_ = c.Crawl(cb)

}

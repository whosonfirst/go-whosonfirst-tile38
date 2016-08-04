package main

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	gabs "github.com/jeffail/gabs"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-geojson"
	"log"
	"os"
	"runtime"
	"strconv"
)

func main() {

	root := flag.String("root", "", "...")
	procs := flag.Int("procs", 200, "...")
	collection := flag.String("collection", "whosonfirst", "...")
	nfs_kludge := flag.Bool("nfs-kludge", false, "Enable the (walk.go) NFS kludge to ignore 'readdirent: errno' 523 errors")

	// verbose := flag.Bool("verbose", false, "...")

	tile38_host := flag.String("tile38-host", "localhost", "...")
	tile38_port := flag.Int("tile38-port", 9851, "...")

	flag.Parse()

	runtime.GOMAXPROCS(*procs)

	tile38_endpoint := fmt.Sprintf("%s:%d", *tile38_host, *tile38_port)
	log.Println("connect to", tile38_endpoint)

	cb := func(abs_path string, info os.FileInfo) error {

		// please put me in a package specific function
		log.Println("index", abs_path)

		feature, err := geojson.UnmarshalFile(abs_path)

		if err != nil {
			log.Printf("PARSE error %v\n", err)
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
			log.Printf("CONNECT error %v\n", err)
			return nil
		}

		defer conn.Close()

		/*
			cmd := "SET"
			args:= make([]interface{}, 0)

			args = append(args, *collection)
			args = append(args, str_wofid)
			args = append(args, "OBJECT")
			args = append(args, str_geom)
		*/

		h := MakeHierarchy(body)

		country_id, ok := h["country_id"]

		if !ok {
			country_id = -1
		}

		region_id, ok := h["region_id"]

		if !ok {
			region_id = -1
		}

		locality_id, ok := h["locality_id"]

		if !ok {
			locality_id = -1
		}

		_, err = conn.Do("SET", *collection, str_wofid, "FIELD", "country_id", country_id, "FIELD", "region_id", region_id, "FIELD", "locality_id", locality_id, "OBJECT", str_geom)

		if err != nil {
			log.Printf("SET error %v\n", err)
			return err
		}

		// http://tile38.com/commands/set/
		// http://tile38.com/commands/fset/

		placetype := feature.Placetype()
		placetype_key := str_wofid + ":placetype"

		_, err = conn.Do("SET", *collection, placetype_key, "STRING", placetype)

		if err != nil {
			fmt.Printf("FAILED to set placetype on %s because, %v\n", placetype_key, err)
		}

		name := feature.Name()
		name_key := str_wofid + ":name"

		_, err = conn.Do("SET", *collection, name_key, "STRING", name)

		if err != nil {
			fmt.Printf("FAILED to set name on %s because, %v\n", name_key, err)
		}

		return nil
	}

	c := crawl.NewCrawler(*root)
	c.NFSKludge = *nfs_kludge

	_ = c.Crawl(cb)

}

func MakeHierarchy(body *gabs.Container) map[string]int64 {

	h := make(map[string]int64)

	hiers, err := body.Path("properties.wof:hierarchy").Children()

	if err != nil {
		return h
	}

	if len(hiers) == 0 {
		return h
	}

	possible, err := hiers[0].ChildrenMap()

	if err != nil {
		return h
	}

	for ancestor, v := range possible {

		wofid := v.Data().(float64)

		h[ancestor] = int64(wofid)
	}

	return h
}

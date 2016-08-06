package main

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-geojson"
	"github.com/whosonfirst/go-whosonfirst-placetypes"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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

	tile38_endpoint := fmt.Sprintf("%s:%d", *tile38_host, *tile38_port)
	log.Println("connect to", tile38_endpoint)

	placetypes, err := placetypes.Init()

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

		// log.Println("index", abs_path)

		feature, err := geojson.UnmarshalFile(abs_path)

		if err != nil {
			log.Printf("PARSE error for %s %v\n", abs_path, err)
			return nil
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

		placetype := feature.Placetype()

		pt, err := placetypes.GetPlacetypeByName(placetype)

		if err != nil {
			log.Println("invalid placetype", placetype)
			return nil
		}

		if *collection == "" {
			*collection = "whosonfirst-" + placetype
		}

		/*

			Basically to do any kind of string/pattern matching with Tile38 we need to encode the value as part
			of the key name so that we can glob it out with a 'MATCH' filter. So the rule of thumb is wof_id + "#" +
			placetype and so on.

			INTERSECTS whosonfirst MATCH *neighbourhood NOFIELDS BOUNDS 40.744152 -73.990474 40.744152 -73.990474
			INTERSECTS whosonfirst MATCH *neigh*ood BOUNDS 40.744152 -73.990474 40.744152 -73.990474

		*/

		repo, ok := feature.StringProperty("wof:repo")

		if !ok {
			log.Println("can't find wof:repo for", str_wofid)
			return nil
		}

		if repo == "" {
			log.Println("missing wof:repo for", str_wofid)
			return nil
		}

		key := str_wofid + "#" + repo

		/*

			The conn.Do method takes a string command and then a "..." of interface{} thingies
			but unfortunately I don't know how to define the latter as a []interface{} and then
			pass that list in so that the compiler thinks they are "..." -able. Good times...
			(20160804/thisisaaronland)

		*/

		/*
			FIELDS are really only good for numeric things that you want to query with a range or that
			you want/need to include with every response item (like wof:id)
		*/

		// log.Println("SET", *collection, key, "FIELD", "wof:id", wofid, "FIELD", "wof:placetype_id", pt.Id, "OBJECT", "<geometry>")

		_, err = conn.Do("SET", *collection, key, "FIELD", "wof:id", wofid, "FIELD", "wof:placetype_id", pt.Id, "OBJECT", str_geom)

		if err != nil {
			log.Printf("SET error %v\n", err)
			return err
		}

		name := feature.Name()
		name_key := str_wofid + ":name"

		_, err = conn.Do("SET", *collection, name_key, "STRING", name)

		if err != nil {
			fmt.Printf("FAILED to set name on %s because, %v\n", name_key, err)
		}

		hiers := body.Path("properties.wof:hierarchy")
		str_hiers := hiers.String()

		hiers_key := str_wofid + ":hierarchy"

		_, err = conn.Do("SET", *collection, hiers_key, "STRING", str_hiers)

		if err != nil {
			fmt.Printf("FAILED to set name on %s because, %v\n", hiers_key, err)
		}

		return nil
	}

	c := crawl.NewCrawler(*root)
	c.NFSKludge = *nfs_kludge

	_ = c.Crawl(cb)

}

// this isn't being used anywhere but it's handy code we should put... somewhere

/*

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

		var wofid int

		fl_wofid, ok := v.Data().(float64)

		if ok {
			wofid = int(fl_wofid)

		} else {

			str_wofid, ok := v.Data().(string)

			if ok {
				wofid, err = strconv.Atoi(str_wofid)

				if err != nil {
					ok = false
				}
			}
		}

		if !ok {
			continue
		}

		h[ancestor] = int64(wofid)
	}

	return h
}

*/

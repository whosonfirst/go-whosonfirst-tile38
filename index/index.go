package tile38

import (
	"bufio"
	"errors"
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
	"sync"
)

type Tile38Client struct {
	Endpoint   string
	Placetypes *placetypes.WOFPlacetypes
	Debug      bool
}

func NewTile38Client(host string, port int) (*Tile38Client, error) {

	pt, err := placetypes.Init()

	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s:%d", host, port)

	client := Tile38Client{
		Endpoint:   endpoint,
		Placetypes: pt,
		Debug:      false,
	}

	conn, err := redis.Dial("tcp", client.Endpoint)

	if err != nil {
		return nil, err
	}

	defer conn.Close()

	rsp, err := conn.Do("PING")

	if err != nil {
		return nil, err
	}

	return &client, nil
}

func (client *Tile38Client) IndexFile(abs_path string, collection string) error {

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

	conn, err := redis.Dial("tcp", client.Endpoint)

	if err != nil {
		return err
	}

	defer conn.Close()

	placetype := feature.Placetype()

	pt, err := client.Placetypes.GetPlacetypeByName(placetype)

	if err != nil {
		return err
	}

	if collection == "" {
		collection = "whosonfirst-" + placetype
	}

	/*

		Basically to do any kind of string/pattern matching with Tile38 we need to encode the value as part
		of the key name so that we can glob it out with a 'MATCH' filter. So the rule of thumb is wof_id + "#" +
		repository and so on.

		INTERSECTS whosonfirst MATCH *neighbourhood NOFIELDS BOUNDS 40.744152 -73.990474 40.744152 -73.990474
		INTERSECTS whosonfirst MATCH *neigh*ood BOUNDS 40.744152 -73.990474 40.744152 -73.990474

	*/

	repo, ok := feature.StringProperty("wof:repo")

	if !ok {
		msg := fmt.Sprintf("can't find wof:repo for %s", str_wofid)
		return errors.New(msg)
	}

	if repo == "" {
		msg := fmt.Sprintf("missing wof:repo for %s", str_wofid)
		return errors.New(msg)
	}

	key := str_wofid + "#" + repo

	/*

		The conn.Do method takes a string command and then a "..." of interface{} thingies
		but unfortunately I don't know how to define the latter as a []interface{} and then
		pass that list in so that the compiler thinks they are "..." -able. Good times...
		(20160804/thisisaaronland)

		FIELDS are really only good for numeric things that you want to query with a range or that
		you want/need to include with every response item (like wof:id)
		(20160807/thisisaaronland)

	*/

	if client.Debug {
		log.Println("SET", collection, key, "FIELD", "wof:id", wofid, "FIELD", "wof:placetype_id", pt.Id, "OBJECT", "...")
		return nil
	}

	_, err = conn.Do("SET", collection, key, "FIELD", "wof:id", wofid, "FIELD", "wof:placetype_id", pt.Id, "OBJECT", str_geom)

	if err != nil {
		return err
	}

	name := feature.Name()
	name_key := str_wofid + ":name"

	_, err = conn.Do("SET", collection, name_key, "STRING", name)

	if err != nil {
		fmt.Printf("FAILED to set name on %s because, %v\n", name_key, err)
	}

	/*
		hiers := body.Path("properties.wof:hierarchy")
		str_hiers := hiers.String()

		hiers_key := str_wofid + ":hierarchy"

		_, err = conn.Do("SET", *collection, hiers_key, "STRING", str_hiers)

		if err != nil {
			fmt.Printf("FAILED to set name on %s because, %v\n", hiers_key, err)
		}
	*/

	return nil

}

func (client *Tile38Client) IndexDirectory(abs_path string, collection string, nfs_kludge bool) error {

	re_wof, _ := regexp.Compile(`(\d+)\.geojson$`)

	cb := func(abs_path string, info os.FileInfo) error {

		// please make me more like this...
		// https://github.com/whosonfirst/py-mapzen-whosonfirst-utils/blob/master/mapzen/whosonfirst/utils/__init__.py#L265

		fname := filepath.Base(abs_path)

		if !re_wof.MatchString(fname) {
			// log.Println("skip", abs_path)
			return nil
		}

		err := client.IndexFile(abs_path, collection)

		if err != nil {
			msg := fmt.Sprintf("failed to index %s, because %v", abs_path, err)
			return errors.New(msg)
		}

		return nil
	}

	c := crawl.NewCrawler(abs_path)
	c.NFSKludge = nfs_kludge

	return c.Crawl(cb)
}

func (client *Tile38Client) IndexFileList(abs_path string, collection string) error {

	file, err := os.Open(abs_path)

	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	count := runtime.GOMAXPROCS(0) // perversely this is how we get the count...
	ch := make(chan bool, count)

	go func() {
		for i := 0; i < count; i++ {
			ch <- true
		}
	}()

	wg := new(sync.WaitGroup)

	for scanner.Scan() {

		<-ch

		path := scanner.Text()

		wg.Add(1)

		go func(path string, collection string, wg *sync.WaitGroup, ch chan bool) {

			defer wg.Done()

			client.IndexFile(path, collection)
			ch <- true

		}(path, collection, wg, ch)
	}

	wg.Wait()

	return nil
}

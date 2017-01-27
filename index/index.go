package tile38

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-csv"
	"github.com/whosonfirst/go-whosonfirst-geojson"
	"github.com/whosonfirst/go-whosonfirst-placetypes"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type Meta struct {
	Name      string           `json:"wof:name"`
	Country   string           `json:"wof:country"`
	// Hierarchy []map[string]int `json:"wof:hierarchy"`
}

type Coords []float64

type Polygon []Coords

type Geometry struct {
	Type        string `json:"type"`
	Coordinates Coords `json:"coordinates"`
}

type GeometryPoly struct {
	Type        string    `json:"type"`
	Coordinates []Polygon `json:"coordinates"`
}

type Tile38Client struct {
	Endpoint   string
	Geometry   string
	Placetypes *placetypes.WOFPlacetypes
	Debug      bool
	Verbose    bool
	pool       *redis.Pool
}

func NewTile38Client(host string, port int) (*Tile38Client, error) {

	pt, err := placetypes.Init()

	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s:%d", host, port)

	// because this:
	// https://github.com/whosonfirst/go-whosonfirst-tile38/issues/8

	tries := 0
	max_tries := 5

	for tries < max_tries {

		tries += 1

		conn, err := redis.Dial("tcp", endpoint)

		if err != nil {
			time.Sleep(time.Second * 1)
			continue
		}

		defer conn.Close()

		_, err = conn.Do("PING")

		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	// https://stackoverflow.com/questions/37828284/redigo-getting-dial-tcp-connect-cannot-assign-requested-address
	// https://godoc.org/github.com/garyburd/redigo/redis#NewPool

	pool := &redis.Pool{
		MaxActive: 1000,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", endpoint)
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}

	client := Tile38Client{
		Endpoint:   endpoint,
		Placetypes: pt,
		Geometry:   "", // use the default geojson geometry
		Debug:      false,
		pool:       pool,
	}

	return &client, nil
}

func (client *Tile38Client) IndexFile(abs_path string, collection string) error {

	// check to see if this is an alt file
	// https://github.com/whosonfirst/go-whosonfirst-tile38/issues/1

	feature, err := geojson.UnmarshalFile(abs_path)

	if err != nil {
		return err
	}

	return client.IndexFeature(feature, collection)
}

func (client *Tile38Client) IndexFeature(feature *geojson.WOFFeature, collection string) error {

	wofid := feature.Id()
	str_wofid := strconv.Itoa(wofid)

	body := feature.Body()

	var str_geom string

	if client.Geometry == "" {

		geom := body.Path("geometry")
		str_geom = geom.String()

	} else if client.Geometry == "bbox" {

		/*

			This is not really the best way to deal with the problem since
			we'll end up with an oversized bounding box. A better way would
			be to store the bounding box for each polygon in the geom and
			flag that in the key name. Which is easy but just requires tweaking
			a few things and really I just want to see if this works at all
			from a storage perspective right now (20160902/thisisaaronland)

		*/

		var swlon float64
		var swlat float64
		var nelon float64
		var nelat float64

		children, _ := body.S("bbox").Children()

		swlon = children[0].Data().(float64)
		swlat = children[1].Data().(float64)
		nelon = children[2].Data().(float64)
		nelat = children[3].Data().(float64)

		poly := Polygon{
			Coords{swlon, swlat},
			Coords{swlon, nelat},
			Coords{nelon, nelat},
			Coords{nelon, swlat},
			Coords{swlon, swlat},
		}

		polys := []Polygon{
			poly,
		}

		geom := GeometryPoly{
			Type:        "Polygon",
			Coordinates: polys,
		}

		bytes, err := json.Marshal(geom)

		if err != nil {
			return err
		}

		str_geom = string(bytes)

	} else if client.Geometry == "centroid" {

		// sudo put me in go-whosonfirst-geojson?
		// (20160829/thisisaaronland)

		var lat float64
		var lon float64
		var lat_ok bool
		var lon_ok bool

		lat, lat_ok = body.Path("properties.lbl:latitude").Data().(float64)
		lon, lon_ok = body.Path("properties.lbl:longitude").Data().(float64)

		if !lat_ok || !lon_ok {

			lat, lat_ok = body.Path("properties.geom:latitude").Data().(float64)
			lon, lon_ok = body.Path("properties.geom:longitude").Data().(float64)
		}

		if !lat_ok || !lon_ok {
			return errors.New("can't find centroid")
		}

		coords := Coords{lon, lat}

		geom := Geometry{
			Type:        "Point",
			Coordinates: coords,
		}

		bytes, err := json.Marshal(geom)

		if err != nil {
			return err
		}

		str_geom = string(bytes)

	} else {

		return errors.New("unknown geometry filter")
	}

	conn := client.pool.Get()
	defer conn.Close()

	// log.Println("number of active connections", client.pool.ActiveCount())

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

	parent, ok := feature.IntProperty("wof:parent_id")

	if !ok {
		log.Printf("FAILED to determine parent ID for %s\n", key)
		parent = -1
	}

	is_superseded := 0
	is_deprecated := 0

	if feature.Deprecated() {
		is_deprecated = 1
	}

	if feature.Superseded() {
		is_superseded = 1
	}

	/*

		The conn.Do method takes a string command and then a "..." of interface{} thingies
		but unfortunately I don't know how to define the latter as a []interface{} and then
		pass that list in so that the compiler thinks they are "..." -able. Good times...
		(20160804/thisisaaronland)

		FIELDS are really only good for numeric things that you want to query with a range or that
		you want/need to include with every response item (like wof:id)
		(20160807/thisisaaronland)

	*/

	if client.Verbose {

		if client.Geometry == "" {
			log.Println("SET", collection, key, "FIELD", "wof:id", wofid, "FIELD", "wof:placetype_id", pt.Id, "FIELD", "wof:parent_id", parent, "FIELD", "wof:is_superseded", is_superseded, "FIELD", "wof:is_deprecated", is_deprecated, "OBJECT", "...")
		} else {
			log.Println("SET", collection, key, "FIELD", "wof:id", wofid, "FIELD", "wof:placetype_id", pt.Id, "FIELD", "wof:parent_id", parent, "FIELD", "wof:is_superseded", is_superseded, "FIELD", "wof:is_deprecated", is_deprecated, "OBJECT", str_geom)
		}

	}

	if !client.Debug {

		_, err := conn.Do("SET", collection, key, "FIELD", "wof:id", wofid, "FIELD", "wof:placetype_id", pt.Id, "FIELD", "wof:parent_id", parent, "FIELD", "wof:is_superseded", is_superseded, "FIELD", "wof:is_deprecated", is_deprecated, "OBJECT", str_geom)

		if err != nil {
			return err
		}
	}

	// https://github.com/whosonfirst/whosonfirst-www-api/blob/master/www/include/lib_whosonfirst_spatial.php#L160
	// When in doubt check here... also, please make me configurable... maybe? (20161017/thisisaaroland)

	meta_key := str_wofid + "#meta"

	name := feature.Name()
	country, ok := feature.StringProperty("wof:country")

	if !ok {
		log.Printf("FAILED to determine country for %s\n", meta_key)
		country = "XX"
	}

	// hier := feature.Hierarchy()

	meta := Meta{
		Name:      name,
		Country:   country,
		// Hierarchy: hier,
	}

	meta_json, err := json.Marshal(meta)

	if err != nil {
		log.Printf("FAILED to marshal JSON on %s because, %v\n", meta_key, err)
		return err
	}

	// See the way we are assigning the meta information to the same collection as the spatial
	// information? We may not always do that (maybe should never do that) but today we do do
	// that... (20161017/thisisaaronland)

	if client.Verbose {
		log.Println("SET", collection, meta_key, "STRING", string(meta_json))
	}

	if !client.Debug {

		_, err := conn.Do("SET", collection, meta_key, "STRING", string(meta_json))

		if err != nil {
			log.Printf("FAILED to set meta on %s because, %v\n", meta_key, err)
			return err
		}
	}

	if client.Verbose {
		log.Println("OKAY", key, meta_key)
	}

	return nil

}

func (client *Tile38Client) IndexMetaFile(csv_path string, collection string, data_root string) error {

	reader, err := csv.NewDictReaderFromPath(csv_path)

	if err != nil {
		return err
	}

	count := runtime.GOMAXPROCS(0) // perversely this is how we get the count...
	ch := make(chan bool, count)

	go func() {
		for i := 0; i < count; i++ {
			ch <- true
		}
	}()

	wg := new(sync.WaitGroup)

	for {
		row, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		rel_path, ok := row["path"]

		if !ok {
			msg := fmt.Sprintf("missing 'path' column in meta file")
			return errors.New(msg)
		}

		abs_path := filepath.Join(data_root, rel_path)

		<-ch

		wg.Add(1)

		go func(ch chan bool) {

			defer func() {
				wg.Done()
				ch <- true
			}()

			client.IndexFile(abs_path, collection)

		}(ch)
	}

	wg.Wait()

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

package index

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-csv"
	"github.com/whosonfirst/go-whosonfirst-geojson"
	"github.com/whosonfirst/go-whosonfirst-placetypes"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"github.com/whosonfirst/go-whosonfirst-tile38/util"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Meta struct {
	Name    string `json:"wof:name"`
	Country string `json:"wof:country"`
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

type Tile38Indexer struct {
	Geometry string
	Debug    bool
	Verbose  bool
	client   tile38.Tile38Client
}

func NewTile38Indexer(client tile38.Tile38Client) (*Tile38Indexer, error) {

	idx := Tile38Indexer{
		Geometry: "", // use the default geojson geometry
		Debug:    false,
		client:   client,
	}

	return &idx, nil
}

func (idx *Tile38Indexer) IndexFile(abs_path string, collection string) error {

	// check to see if this is an alt file
	// https://github.com/whosonfirst/go-whosonfirst-tile38/issues/1

	feature, err := geojson.UnmarshalFile(abs_path)

	if err != nil {
		return err
	}

	return idx.IndexFeature(feature, collection)
}

func (idx *Tile38Indexer) IndexFeature(feature *geojson.WOFFeature, collection string) error {

	wofid := feature.Id()
	str_wofid := strconv.Itoa(wofid)

	placetype := feature.Placetype()

	body := feature.Body()

	var str_geom string

	if idx.Geometry == "" {

		geom := body.Path("geometry")
		str_geom = geom.String()

	} else if idx.Geometry == "bbox" {

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

	} else if idx.Geometry == "centroid" {

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

	pt, err := placetypes.GetPlacetypeByName(placetype)

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

	set_cmd := []string{
		"SET", collection, key,
		"FIELD", "wof:id", strconv.Itoa(wofid),
		"FIELD", "wof:placetype_id", strconv.FormatInt(pt.Id, 10),
		"FIELD", "wof:parent_id", strconv.Itoa(parent),
		"FIELD", "wof:is_superseded", strconv.Itoa(is_superseded),
		"FIELD", "wof:is_deprecated", strconv.Itoa(is_deprecated),
		"OBJECT", str_geom,
	}

	if idx.Verbose {

		// make a copy in case we don't want to print out the entire geom
		// and we're not running in debug mode in which case we'll, you
		// know... need the geom (20170305/thisisaaronland)

		set_cmd_copy := set_cmd

		if idx.Geometry == "" {
			set_cmd_copy[len(set_cmd_copy)-1] = "..."
		}

		log.Println(strings.Join(set_cmd_copy, " "))
	}

	if !idx.Debug {

		t38_cmd, t38_args := util.ListToRESPCommand(set_cmd)
		_, err := idx.client.Do(t38_cmd, t38_args...)

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
		Name:    name,
		Country: country,
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

	meta_cmd := []string{
		"SET", collection, meta_key,
		"STRING", string(meta_json),
	}

	if idx.Verbose {
		log.Println(strings.Join(meta_cmd, " "))
	}

	if !idx.Debug {

		t38_cmd, t38_args := util.ListToRESPCommand(meta_cmd)

		_, err := idx.client.Do(t38_cmd, t38_args...)

		if err != nil {
			log.Printf("FAILED to set meta on %s because, %v\n", meta_key, err)
			return err
		}
	}

	if idx.Verbose {
		log.Println("OKAY", key, meta_key)
	}

	return nil

}

func (idx *Tile38Indexer) IndexMetaFile(csv_path string, collection string, data_root string) error {

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

			if !idx.EnsureWOF(abs_path, false) {
				return
			}

			idx.IndexFile(abs_path, collection)

		}(ch)
	}

	wg.Wait()

	return nil
}

func (idx *Tile38Indexer) IndexDirectory(abs_path string, collection string, nfs_kludge bool) error {

	cb := func(abs_path string, info os.FileInfo) error {

		if !idx.EnsureWOF(abs_path, false) {
			return nil
		}

		err := idx.IndexFile(abs_path, collection)

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

func (idx *Tile38Indexer) IndexFileList(abs_path string, collection string) error {

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

		go func(abs_path string, collection string, wg *sync.WaitGroup, ch chan bool) {

			defer wg.Done()

			if !idx.EnsureWOF(abs_path, false) {
				return
			}

			idx.IndexFile(abs_path, collection)
			ch <- true

		}(path, collection, wg, ch)
	}

	wg.Wait()

	return nil
}

func (idx *Tile38Indexer) EnsureWOF(abs_path string, allow_alt bool) bool {

	wof, err := uri.IsWOFFile(abs_path)

	if err != nil {
		log.Println(fmt.Sprintf("Failed to determine whether %s is a WOF file, because %s", abs_path, err))
		return false
	}

	if !wof {
		return false
	}

	alt, err := uri.IsAltFile(abs_path)

	if err != nil {
		log.Println(fmt.Sprintf("Failed to determine whether %s is an alt file, because %s", abs_path, err))
		return false
	}

	if alt && !allow_alt {
		return false
	}

	return true
}

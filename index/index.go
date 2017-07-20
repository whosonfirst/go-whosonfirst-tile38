package index

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-csv"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/geojson"
	geom "github.com/whosonfirst/go-whosonfirst-geojson-v2/properties/geometry"
	wof "github.com/whosonfirst/go-whosonfirst-geojson-v2/properties/whosonfirst"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/whosonfirst"
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

	feature, err := whosonfirst.LoadFeatureFromFile(abs_path)

	if err != nil {
		return err
	}

	return idx.IndexFeature(feature, collection)
}

func (idx *Tile38Indexer) IndexFeature(feature geojson.Feature, collection string) error {

	wofid := wof.Id(feature)
	str_wofid := strconv.FormatInt(wofid, 10)

	parent_id := wof.ParentId(feature)
	str_parent_id := strconv.FormatInt(parent_id, 10)

	name := wof.Name(feature)
	country := wof.Country(feature)

	repo := wof.Repo(feature)

	if repo == "" {
		msg := fmt.Sprintf("missing wof:repo for %s", str_wofid)
		return errors.New(msg)
	}

	placetype := wof.Placetype(feature)

	pt, err := placetypes.GetPlacetypeByName(placetype)

	if err != nil {
		return err
	}

	str_placetype_id := strconv.FormatInt(pt.Id, 10)

	str_superseded := "0"
	str_deprecated := "0"

	if wof.IsDeprecated(feature) {
		str_deprecated = "1"
	}

	if wof.IsSuperseded(feature) {
		str_superseded = "1"
	}

	geom_key := str_wofid + "#" + repo
	meta_key := str_wofid + "#meta"

	var str_geom string

	if idx.Geometry == "" {

		s, err := geom.ToString(feature)

		if err != nil {
			return err
		}

		str_geom = s

	} else if idx.Geometry == "bbox" {

		bboxes, err := feature.BoundingBoxes()

		if err != nil {
			return err
		}

		mbr := bboxes.MBR()
		sw := mbr.Min
		ne := mbr.Max

		poly := Polygon{
			Coords{sw.X, sw.Y},
			Coords{sw.X, ne.Y},
			Coords{ne.X, ne.Y},
			Coords{ne.X, sw.Y},
			Coords{sw.X, sw.Y},
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

		lat, lon := wof.Centroid(feature)

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

	set_cmd := "SET"

	set_args := []interface{}{
		collection, geom_key,
		"FIELD", "wof:id", str_wofid,
		"FIELD", "wof:placetype_id", str_placetype_id,
		"FIELD", "wof:parent_id", str_parent_id,
		"FIELD", "wof:is_superseded", str_superseded,
		"FIELD", "wof:is_deprecated", str_deprecated,
		"OBJECT", str_geom,
	}

	if idx.Verbose {

		// make a copy in case we don't want to print out the entire geom
		// and we're not running in debug mode in which case we'll, you
		// know... need the geom (20170305/thisisaaronland)

		if idx.Geometry != "" {
			log.Println(util.RESPCommandToString(set_cmd, set_args))
		} else {

			copy_args := make([]interface{}, 0)

			count := len(set_args)
			last := count - 2

			for _, a := range set_args[:last] {
				copy_args = append(copy_args, a)
			}

			copy_args = append(copy_args, "...")

			log.Println(util.RESPCommandToString(set_cmd, copy_args))
		}

	}

	if !idx.Debug {

		rsp, err := idx.client.Do(set_cmd, set_args...)

		if err != nil {
			return err
		}

		err = util.EnsureOk(rsp)

		if err != nil {
			log.Printf("FAILED to SET key for %s because, %v\n", geom_key, err)
			return err
		}
	}

	// https://github.com/whosonfirst/whosonfirst-www-api/blob/master/www/include/lib_whosonfirst_spatial.php#L160
	// When in doubt check here... also, please make me configurable... maybe? (20161017/thisisaaroland)

	meta := Meta{
		Name:    name,
		Country: country,
	}

	meta_json, err := json.Marshal(meta)

	if err != nil {
		log.Printf("FAILED to marshal JSON on %s because, %v\n", meta_key, err)
		return err
	}

	// See the way we are assigning the meta information to the same collection as the spatial
	// information? We may not always do that (maybe should never do that) but today we do do
	// that... (20161017/thisisaaronland)

	meta_cmd := "SET"

	meta_args := []interface{}{
		collection, meta_key,
		"STRING", string(meta_json),
	}

	if idx.Verbose {
		log.Println(util.RESPCommandToString(meta_cmd, meta_args))
	}

	if !idx.Debug {

		rsp, err := idx.client.Do(meta_cmd, meta_args...)

		if err != nil {
			log.Printf("FAILED to SET key for %s because, %v\n", meta_key, err)
			return err
		}

		err = util.EnsureOk(rsp)

		if err != nil {
			log.Printf("FAILED to SET key for %s because, %v\n", meta_key, err)
			return err
		}
	}

	if idx.Verbose {
		log.Println("SET", geom_key)
		log.Println("SET", meta_key)
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

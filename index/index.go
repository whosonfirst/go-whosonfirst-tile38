package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/properties/geometry"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/properties/whosonfirst"
	"github.com/whosonfirst/go-whosonfirst-placetypes"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"github.com/whosonfirst/go-whosonfirst-tile38/util"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"log"
	"strconv"
)

// see notes inre go-whosonfirst-spr below

type Meta struct {
	Name    string `json:"wof:name"`
	Country string `json:"wof:country"`
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
	Strict   bool
	clients  []tile38.Tile38Client
}

func NewTile38Indexer(clients ...tile38.Tile38Client) (*Tile38Indexer, error) {

	idx := Tile38Indexer{
		Geometry: "", // use the default geojson geometry
		Debug:    false,
		Verbose:  false,
		Strict:   true,
		clients:  clients,
	}

	return &idx, nil
}

func (idx *Tile38Indexer) IndexFeature(feature geojson.Feature, collection string) error {

	wofid := whosonfirst.Id(feature)
	str_wofid := strconv.FormatInt(wofid, 10)

	parent_id := whosonfirst.ParentId(feature)
	str_parent_id := strconv.FormatInt(parent_id, 10)

	name := whosonfirst.Name(feature)
	country := whosonfirst.Country(feature)

	repo := whosonfirst.Repo(feature)

	if repo == "" {
		msg := fmt.Sprintf("missing wof:repo for %s", str_wofid)
		return errors.New(msg)
	}

	placetype := whosonfirst.Placetype(feature)

	pt, err := placetypes.GetPlacetypeByName(placetype)

	if err != nil {
		return err
	}

	str_placetype_id := strconv.FormatInt(pt.Id, 10)

	is_current, err := whosonfirst.IsCurrent(feature)

	if err != nil {
		return err
	}

	is_deprecated, err := whosonfirst.IsDeprecated(feature)

	if err != nil {
		return err
	}

	is_ceased, err := whosonfirst.IsCeased(feature)

	if err != nil {
		return err
	}

	is_superseded, err := whosonfirst.IsSuperseded(feature)

	if err != nil {
		return err
	}

	is_superseding, err := whosonfirst.IsSuperseding(feature)

	if err != nil {
		return err
	}

	// log.Printf("existential current: %s ceased: %s deprecated: %s superseded: %s\n", str_current, str_ceased, str_deprecated, str_superseded)

	geom_key := str_wofid + "#" + repo
	meta_key := str_wofid + "#meta"

	var str_geom string

	if idx.Geometry == "" {

		// log.Printf("%s derived geometry from string\n", geom_key)

		s, err := geometry.ToString(feature)

		if err != nil {
			return err
		}

		str_geom = s

	} else if idx.Geometry == "bbox" {

		// log.Printf("%s derived geometry from bounding box\n", geom_key)

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

		// log.Printf("%s derived geometry from centroid\n", geom_key)

		centroid, err := whosonfirst.Centroid(feature)

		if err != nil {
			return err
		}

		c := centroid.Coord()

		coords := Coords{c.X, c.Y}

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

	str_current := is_current.StringFlag()
	str_deprecated := is_deprecated.StringFlag()
	str_ceased := is_ceased.StringFlag()
	str_superseded := is_superseded.StringFlag()
	str_superseding := is_superseding.StringFlag()

	set_cmd := "SET"

	set_args := []interface{}{
		collection, geom_key,
		"FIELD", "wof:id", str_wofid,
		"FIELD", "wof:placetype_id", str_placetype_id,
		"FIELD", "wof:parent_id", str_parent_id,
		"FIELD", "mz:is_current", str_current,
		"FIELD", "mz:is_deprecated", str_deprecated,
		"FIELD", "mz:is_ceased", str_ceased,
		"FIELD", "mz:is_superseded", str_superseded,
		"FIELD", "mz:is_superseding", str_superseding,
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

		err := idx.Do(set_cmd, set_args...)

		if err != nil {
			log.Printf("FAILED to SET (geom) key for %s because, %v\n", geom_key, err)
			// log.Println(set_args)
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

	// TBD... just doing this instead of asking T38 consumers to rebuild responses
	// based on a combination of geom + string keys... (20170731/thisisaaronland)

	// https://github.com/whosonfirst/go-whosonfirst-spr - THIS IS NOT READY FOR USE YET
	// import spr "github.com/whosonfirst/go-whosonfirst-spr/whosonfirst"
	// meta, err := spr.NewSPRFromFeature(feature)
	// meta_json, err := json.Marshal(meta)

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

		err := idx.Do(meta_cmd, meta_args...)

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

func (idx *Tile38Indexer) Do(cmd string, args ...interface{}) error {

	err_ch := make(chan error)
	done_ch := make(chan bool)

	for _, c := range idx.clients {

		go func(err_ch chan error, done_ch chan bool, c tile38.Tile38Client, cmd string, args ...interface{}) {

			defer func() {
				done_ch <- true
			}()

			rsp, err := c.Do(cmd, args...)

			if err != nil {
				msg := fmt.Sprintf("FAILED issuing command to client (%s) because, %v", c.Endpoint(), err)
				err_ch <- errors.New(msg)
				return
			}

			err = util.EnsureOk(rsp)

			if err != nil {
				msg := fmt.Sprintf("FAILED to ensure ok for command to client (%s) for %s because, %v\n", c.Endpoint(), err)
				err_ch <- errors.New(msg)
				return
			}

		}(err_ch, done_ch, c, cmd, args...)

	}

	pending := len(idx.clients)

	for pending > 0 {

		select {
		case err := <-err_ch:
			return err
		case <-done_ch:
			pending -= 1
		}
	}

	return nil
}

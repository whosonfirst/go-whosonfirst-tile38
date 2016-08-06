package tile38

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/whosonfirst/go-whosonfirst-geojson"
	"github.com/whosonfirst/go-whosonfirst-placetypes"
	"strconv"
)

type Tile38Client struct {
	endpoint   string
	placetypes *placetypes.WOFPlacetypes
}

func NewTile38Client(host string, port int) (*Tile38Client, error) {

	pt, err := placetypes.Init()

	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s:%d", host, port)

	client := Tile38Client{
		endpoint:   endpoint,
		placetypes: pt,
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

	conn, err := redis.Dial("tcp", client.endpoint)

	if err != nil {
		return err
	}

	defer conn.Close()

	placetype := feature.Placetype()

	pt, err := client.placetypes.GetPlacetypeByName(placetype)

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

	fmt.Println("SET", collection, key, "FIELD", "wof:id", wofid, "FIELD", "wof:placetype_id", pt.Id, "OBJECT", "...")
	return nil

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

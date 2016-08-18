package tile38

/*

	This is basically a line-by-line copy of index/index.go with the exception
	of the per-source code to extract coordinate data. This is expedience rather
	than a feature. It remains to be sorted out... (20160818/thisisaaronland)

*/

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	"github.com/whosonfirst/go-whosonfirst-geojson"
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
	Debug      bool
}

func NewTile38Client(host string, port int) (*Tile38Client, error) {

	endpoint := fmt.Sprintf("%s:%d", host, port)

	client := Tile38Client{
		Endpoint:   endpoint,
		Debug:      false,
	}

	conn, err := redis.Dial("tcp", client.Endpoint)

	if err != nil {
		return nil, err
	}

	defer conn.Close()

	_, err = conn.Do("PING")

	if err != nil {
		return nil, err
	}

	return &client, nil
}

func (client *Tile38Client) IndexFile(abs_path string, source string) error {

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

	conn, err := redis.Dial("tcp", client.Endpoint)

	if err != nil {
		return err
	}

	defer conn.Close()

	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s#%s", str_wofid, source)

	var lat float64
	var lon float64

	if source == "whosonfirst" {

		lat = body.Path("properties.geom:latitude").Data().(float64)
		lon = body.Path("properties.geom:longitude").Data().(float64)

	} else {
		
		children, _ := body.S("geometry").ChildrenMap()

		for key, child := range children {

		    if key != "coordinates" {
			continue
		    }	

		    var coords []interface{}
		    coords, _ = child.Data().([]interface{})

		    lon = coords[0].(float64)
		    lat = coords[1].(float64)

		    break
		}
	}

	if client.Debug {
		log.Println("SET", source, key, "FIELD", "id", wofid, "POINT", lat, lon)
		return nil
	}

	_, err = conn.Do("SET", source, key, "FIELD", "id", wofid, "POINT", lat, lon)

	if err != nil {
		return err
	}

	name := feature.Name()
	name_key := fmt.Sprintf("%s#%s:name", str_wofid, source)

	_, err = conn.Do("SET", source, name_key, "STRING", name)

	if err != nil {
		fmt.Printf("FAILED to set name on %s because, %v\n", name_key, err)
	}

	return nil

}

func (client *Tile38Client) IndexDirectory(abs_path string, source string, nfs_kludge bool) error {

	re_wof, _ := regexp.Compile(`(\d+)\.geojson$`)

	cb := func(abs_path string, info os.FileInfo) error {

		// please make me more like this...
		// https://github.com/whosonfirst/py-mapzen-whosonfirst-utils/blob/master/mapzen/whosonfirst/utils/__init__.py#L265

		fname := filepath.Base(abs_path)

		if !re_wof.MatchString(fname) {
			// log.Println("skip", abs_path)
			return nil
		}

		err := client.IndexFile(abs_path, source)

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

func (client *Tile38Client) IndexFileList(abs_path string, source string) error {

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

		go func(path string, source string, wg *sync.WaitGroup, ch chan bool) {

			defer wg.Done()

			client.IndexFile(path, source)
			ch <- true

		}(path, source, wg, ch)
	}

	wg.Wait()

	return nil
}

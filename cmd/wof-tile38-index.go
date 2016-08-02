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

func main(){

	root := flag.String("root", "", "...")
	procs := flag.Int("procs", 200, "...")
	// verbose := flag.Bool("verbose", false, "...")
	
	tile38_host := flag.String("tile38-host", "localhost", "...")
	tile38_port := flag.Int("tile38-port", 9851, "...")

	flag.Parse()

	runtime.GOMAXPROCS(*procs)

	tile38_endpoint := fmt.Sprintf("%s:%d", *tile38_host, *tile38_port)
	fmt.Println("connect to", tile38_endpoint)

	cb := func(abs_path string, info os.FileInfo) error {

		fmt.Println("index", abs_path)

		feature, err := geojson.UnmarshalFile(abs_path)

		if err != nil {
		   fmt.Println("FAIL 1", err)
		   return err		       
		}

		wofid := feature.Id()
		str_wofid := strconv.Itoa(wofid)

		if err != nil {
		   fmt.Println("FAIL 2", err)
		   return err
		}

		body := feature.Body()
		geom := body.Path("geometry")

		str_geom := geom.String()

		conn, err := redis.Dial("tcp", tile38_endpoint)

		if err != nil {
	   	   fmt.Println("FAIL 3", err)
		   return nil
		}

		defer conn.Close()

		_, err = conn.Do("SET", "whosonfirst", str_wofid, "OBJECT", str_geom)

		if err != nil {
		   fmt.Println("FAIL 4", err)
		   return err
		}

		fmt.Println("OK", wofid)
		return nil
	}

	c := crawl.NewCrawler(*root)
	_ = c.Crawl(cb)
     
}
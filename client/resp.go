package client

// This does not work. I can't figure out how/where/what to unserialize the
// RESP response...

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	_ "log"
	"time"
)

type RESPClient struct {
	tile38.Tile38Client
	Endpoint string
	Debug    bool
	Verbose  bool
	pool     *redis.Pool
}

func NewRESPClient(host string, port int) (*RESPClient, error) {

	t38_endpoint := fmt.Sprintf("%s:%d", host, port)

	// because this:
	// https://github.com/whosonfirst/go-whosonfirst-tile38/issues/8

	tries := 0
	max_tries := 5

	var err error

	for tries < max_tries {

		tries += 1

		conn, err := redis.Dial("tcp", t38_endpoint)

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
			c, err := redis.Dial("tcp", t38_endpoint)
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}

	client := RESPClient{
		Endpoint: t38_endpoint,
		Debug:    false,
		pool:     pool,
	}

	return &client, nil
}

func (cl *RESPClient) Do(t38_cmd string, t38_args ...interface{}) (interface{}, error) {

	conn := cl.pool.Get()
	defer conn.Close()

	rsp, err := conn.Do(t38_cmd, t38_args...)

	if err != nil {
		return nil, err
	}

	// this will fail...
	return rsp, nil
}

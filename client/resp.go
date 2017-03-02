package client

import (
       "github.com/whosonfirst/go-whosonfirst-tile38"
	"github.com/garyburd/redigo/redis"
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

	endpoint := fmt.Sprintf("%s:%d", host, port)

	// because this:
	// https://github.com/whosonfirst/go-whosonfirst-tile38/issues/8

	tries := 0
	max_tries := 5

	var err error

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
		Endpoint: endpoint,
		Debug:    false,
		pool:     pool,
	}

	return &client, nil
}

func (cl *RESPClient) Do(cmd string) (error) {

	_, err = conn.Do(cmd)

	if err != nil {
		return nil, err
	}

}

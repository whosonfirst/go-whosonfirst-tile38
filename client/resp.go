package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	_ "log"
	"time"
)

type RESPClient struct {
	tile38.Tile38Client
	endpoint string
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
		MaxActive:   1000,
		MaxIdle:     100,
		IdleTimeout: 10 * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {

			c, err := redis.Dial("tcp", t38_endpoint)

			if err != nil {
				return nil, err
			}

			// because this: https://github.com/tidwall/tile38/issues/153

			json_rsp, err := redis.String(c.Do("OUTPUT", "json"))

			if err != nil {
				msg := fmt.Sprintf("Dial failed because %s", err)
				return nil, errors.New(msg)
			}

			if !gjson.Get(json_rsp, "ok").Bool() {
				return nil, errors.New(gjson.Get(json_rsp, "err").String())
			}

			return c, err
		},
	}

	client := RESPClient{
		endpoint: t38_endpoint,
		pool:     pool,
	}

	return &client, nil
}

func (cl *RESPClient) Endpoint() string {
	return cl.endpoint
}

func (cl *RESPClient) Do(t38_cmd string, t38_args ...interface{}) (interface{}, error) {

	conn := cl.pool.Get()
	defer conn.Close()

	redis_rsp, err := conn.Do(t38_cmd, t38_args...)

	if err != nil {
		return nil, err
	}

	json_rsp, err := redis.Bytes(redis_rsp, nil)

	if err != nil {
		return nil, err
	}

	var t38_rsp tile38.Tile38Response
	err = json.Unmarshal(json_rsp, &t38_rsp)

	if err != nil {
		return nil, err
	}

	return t38_rsp, nil
}

func (cl *RESPClient) DoMeta(t38_cmd string, t38_args ...interface{}) (interface{}, error) {

	conn := cl.pool.Get()
	defer conn.Close()

	redis_rsp, err := conn.Do(t38_cmd, t38_args...)

	if err != nil {
		return nil, err
	}

	json_rsp, err := redis.Bytes(redis_rsp, nil)

	if err != nil {
		return nil, err
	}

	var t38_rsp tile38.Tile38MetaResponse
	err = json.Unmarshal(json_rsp, &t38_rsp)

	if err != nil {
		return nil, err
	}

	return t38_rsp, nil
}

package client

import (
	"encoding/json"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type HTTPClient struct {
	tile38.Tile38Client
	endpoint string
}

func NewHTTPClient(host string, port int) (*HTTPClient, error) {

	endpoint := fmt.Sprintf("%s:%d", host, port)

	client := HTTPClient{
		endpoint: endpoint,
	}

	return &client, nil
}

func (cl *HTTPClient) Endpoint() string {
	return cl.endpoint
}

func (cl *HTTPClient) Do(t38_cmd string, t38_args ...interface{}) (interface{}, error) {

	http_cmd := []string{
		t38_cmd,
	}

	for _, a := range t38_args {
		http_cmd = append(http_cmd, a.(string))
	}

	str_cmd := strings.Join(http_cmd, " ")

	t38_url := fmt.Sprintf("http://%s/%s", cl.endpoint, url.QueryEscape(str_cmd))

	http_rsp, err := http.Get(t38_url)

	if err != nil {
		return nil, err
	}

	defer http_rsp.Body.Close()

	results, err := ioutil.ReadAll(http_rsp.Body)

	if err != nil {
		return nil, err
	}

	var t38_rsp tile38.Tile38Response
	err = json.Unmarshal(results, &t38_rsp)

	if err != nil {
		return nil, err
	}

	return t38_rsp, nil
}

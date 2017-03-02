package client

import (
       "encoding/json"
       "fmt"
       "github.com/whosonfirst/go-whosonfirst-tile38"
       "io/ioutil"
       "net/http"
       "net/url"
)

type HTTPClient struct {
        tile38.Tile38Client
	Endpoint string
	Debug    bool
	Verbose  bool
}

func NewHTTPClient(host string, port int) (*HTTPClient, error) {

     endpoint := fmt.Sprintf("%s:%d", host, port)

	client := HTTPClient{
		Endpoint: endpoint,
		Debug:    false,
	}

	return &client, nil
}

func (cl *HTTPClient) Do(t38_cmd string) (interface{}, error) {

		t38_url := fmt.Sprintf("http://%s/%s", cl.Endpoint, url.QueryEscape(t38_cmd))

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

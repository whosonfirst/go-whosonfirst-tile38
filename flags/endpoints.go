package flags

import (
	"errors"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"github.com/whosonfirst/go-whosonfirst-tile38/client"
	"strconv"
	"strings"
)

type Endpoints []string

func (e *Endpoints) String() string {
	return strings.Join(*e, "\n")
}

func (e *Endpoints) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func (e *Endpoints) ToClients() ([]tile38.Tile38Client, error) {

	clients := make([]tile38.Tile38Client, 0)

	for _, str_pair := range *e {

		pair := strings.Split(str_pair, ":")

		if len(pair) > 2 {
			msg := fmt.Sprintf("Invalid endpoint string %s", str_pair)
			return nil, errors.New(msg)
		}

		var host string
		var port int

		if len(pair) == 1 {
			host = pair[0]
			port = 9851
		} else {

			p, err := strconv.Atoi(pair[1])

			if err != nil {
				return nil, err
			}

			host = pair[0]
			port = p
		}

		t38_client, err := client.NewRESPClient(host, port)

		if err != nil {
			return nil, err
		}

		clients = append(clients, t38_client)
	}

	return clients, nil
}

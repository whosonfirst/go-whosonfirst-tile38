package util

import (
	"errors"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-tile38"
	"strings"
)

func RESPCommandToString(cmd string, args []interface{}) string {

	str_cmd := []string{cmd}

	for _, a := range args {
		str_cmd = append(str_cmd, a.(string))
	}

	return strings.Join(str_cmd, " ")
}

func EnsureOk(rsp interface{}) error {

	if !rsp.(tile38.Tile38Response).Ok {

		return errors.New(fmt.Sprintf("Tile38 command failed because... computers?"))
	}

	return nil
}

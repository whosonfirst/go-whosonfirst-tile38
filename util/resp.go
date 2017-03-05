package util

import (
	"strings"
)

func StringToRESPCommand(str string) (string, []interface{}) {

	parts := strings.Split(str, " ")
	return ListToRESPCommand(parts)
}

func ListToRESPCommand(list []string) (string, []interface{}) {

	cmd := list[0]
	args := ListToRESPArgs(list[1:])

	return cmd, args
}

func ListToRESPArgs(list []string) []interface{} {

	args := make([]interface{}, 0)

	for _, a := range list {
		args = append(args, a)
	}

	return args
}

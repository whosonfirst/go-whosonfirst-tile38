package util

import (
	"strings"
)

func StringToRESPCommand(str string) (string, []interface{}) {

	parts := strings.Split(str, " ")
	return ListToRESPCommand(parts)
}

func ListToRESPCommand(list []string) (string, []interface{}) {

	chunks := make([]string, 0)

	for _, phrase := range list {

		for _, chars := range strings.Split(phrase, " ") {
			chunks = append(chunks, chars)
		}
	}

	cmd := chunks[0]
	args := ListToRESPArgs(chunks[1:])

	return cmd, args
}

func ListToRESPArgs(list []string) []interface{} {

	args := make([]interface{}, 0)

	for _, chars := range list {
		args = append(args, chars)
	}

	return args
}

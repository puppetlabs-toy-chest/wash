package parser

import "strings"

var defaultPath = "."

func parsePath(args []string) (string, []string) {
	if len(args) == 0 {
		return defaultPath, args
	}
	path := args[0]
	if path == "" {
		return defaultPath, args[1:]
	}
	// This check is a short-hand way of checking if only
	// "expression" was specified in the args
	if strings.Contains("-()", string(path[0])) {
		return defaultPath, args
	}
	// A path was specified, so shift the arguments
	return path, args[1:]
}

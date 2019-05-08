package cmdutil

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

var levelMap = map[string]log.Level{
	"warn":  log.WarnLevel,
	"info":  log.InfoLevel,
	"debug": log.DebugLevel,
	"trace": log.TraceLevel,
}

func ParseLevel(s string) (log.Level, error) {
	if level, ok := levelMap[s]; ok {
		return level, nil
	}

	var allLevels []string
	for level := range levelMap {
		allLevels = append(allLevels, level)
	}

	return log.FatalLevel,
		fmt.Errorf("%v is not a valid level. Valid levels are %v", s, strings.Join(allLevels, ", "))
}

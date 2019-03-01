package cmd

import (
	"fmt"

	"github.com/InVisionApp/tabular"
)

func longestFieldFromColumn(entries [][]string, colIdx int) string {
	max := 0
	var match string
	for _, entry := range entries {
		s := entry[colIdx]
		l := len(s)
		if l > max {
			max = l
			match = s
		}
	}
	return match
}

type columnHeader struct {
	shortName, fullName string
}

func formatTabularListing(headers []columnHeader, entries [][]string) string {
	// Setup the output table
	tab := tabular.New()
	for i, column := range headers {
		width := len(longestFieldFromColumn(entries, i))
		tab.Col(column.shortName, column.fullName, width+2)
	}

	table := tab.Parse("*")
	out := fmt.Sprintln(table.Header)

	values := make([]interface{}, len(headers))
	for _, entry := range entries {
		if len(values) != len(entry) {
			panic("all entries must be the same length")
		}
		for i, item := range entry {
			values[i] = item
		}
		out += fmt.Sprintf(table.Format, values...)
	}
	return out
}

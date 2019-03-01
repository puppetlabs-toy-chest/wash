package cmd

import (
	"fmt"
	"time"

	"github.com/InVisionApp/tabular"
	"github.com/fatih/color"
)

func eprintf(msg string, a ...interface{}) {
	_, err := fmt.Fprintf(color.Error, color.RedString(msg), a...)
	if err != nil {
		panic(err)
	}
}

func formatDuration(dur time.Duration) string {
	const Decisecond = 100 * time.Millisecond
	const Day = 24 * time.Hour
	d := dur / Day
	dur = dur % Day
	h := dur / time.Hour
	dur = dur % time.Hour
	m := dur / time.Minute
	dur = dur % time.Minute
	s := dur / time.Second
	dur = dur % time.Second
	f := dur / Decisecond
	if d >= 1 {
		return fmt.Sprintf("%02d-%02d:%02d:%02d.%02d", d, h, m, s, f)
	} else if h >= 1 {
		return fmt.Sprintf("%02d:%02d:%02d.%02d", h, m, s, f)
	} else {
		return fmt.Sprintf("%02d:%02d.%02d", m, s, f)
	}
}

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
		// Don't pad the last column
		var width int
		if i < len(headers)-1 {
			width = len(longestFieldFromColumn(entries, i)) + 2
		}
		tab.Col(column.shortName, column.fullName, width)
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

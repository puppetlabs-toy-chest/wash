package cmdutil

import (
	"fmt"

	"github.com/InVisionApp/tabular"
)

// LongestFieldFromColumn returns the longest string for a particular column index
// from the provided table.
func LongestFieldFromColumn(rows [][]string, colIdx int) string {
	max := 0
	var match string
	for _, row := range rows {
		s := row[colIdx]
		l := len(s)
		if l > max {
			max = l
			match = s
		}
	}
	return match
}

// ColumnHeader describes a short and long name for a column.
type ColumnHeader struct {
	ShortName, FullName string
}

// FormatTable formats the provided headers and string table to display
// with sufficient padding to align columns.
func FormatTable(headers []ColumnHeader, rows [][]string) string {
	// Setup the output table
	tab := tabular.New()
	for i, column := range headers {
		// Don't pad the last column
		var width int
		if i < len(headers)-1 {
			width = len(LongestFieldFromColumn(rows, i)) + 2
		}
		tab.Col(column.ShortName, column.FullName, width)
	}

	table := tab.Parse("*")
	out := fmt.Sprintln(table.Header)

	values := make([]interface{}, len(headers))
	for _, row := range rows {
		if len(values) != len(row) {
			panic("all rows must be the same length")
		}
		for i, item := range row {
			values[i] = item
		}
		out += fmt.Sprintf(table.Format, values...)
	}
	return out
}

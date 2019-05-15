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

// Table represents a formatted table. Use the NewTable* functions
// to create your Table objects.
type Table struct {
	headers []ColumnHeader
	rows [][]string
	numColumns int
	hasHeaders bool
}

// NewTable creates a new Table object with the given
// rows
func NewTable(rows... []string) *Table {
	if len(rows) == 0 {
		panic("cmdutil.NewTable called without any rows")
	}
	return &Table{
		rows: rows,
		numColumns: len(rows[0]),
		hasHeaders: false,
	}
}

// NewTableWithHeaders creates a new Table object with the given
// headers and rows
func NewTableWithHeaders(headers []ColumnHeader, rows [][]string) *Table {
	if len(headers) == 0 {
		panic("cmdutil.NewTableWithHeaders called without any headers")
	}
	if len(rows) == 0 {
		panic("cmdutil.NewTableWithHeaders called without any rows")
	}
	return &Table{
		headers: headers,
		rows: rows,
		numColumns: len(headers),
		hasHeaders: true,
	}
}

// Rows returns the table's rows
func (t *Table) Rows() [][]string {
	return t.rows
}

// Append appends the given rows to t
func (t *Table) Append(rows [][]string) {
	t.rows = append(t.rows, rows...)
}

// Format formats the provided table to display with sufficient padding
// to align columns
func (t *Table) Format() string {
	headers := make([]ColumnHeader, t.numColumns)
	if t.hasHeaders {
		headers = t.headers
	}

	// Setup the output table
	tab := tabular.New()
	for i, column := range headers {
		// Don't pad the last column
		var width int
		if i < len(headers)-1 {
			longestColumn := LongestFieldFromColumn(t.rows, i)
			if !t.hasHeaders {
				// Construct a stub header so that the tabular
				// library generates the right format string
				// for the table. Using the zeroed ColumnHeader
				// objects is not enough because they still result
				// in an incorrectly formatted table.
				column.ShortName = longestColumn
				column.FullName = longestColumn
			}
			width = len(longestColumn) + 2
		}
		tab.Col(column.ShortName, column.FullName, width)
	}

	table := tab.Parse("*")
	out := ""
	if t.hasHeaders {
		out = fmt.Sprintln(table.Header)
	}

	values := make([]interface{}, t.numColumns)
	for _, row := range t.rows {
		if len(row) != t.numColumns {
			panic("all rows must have the same number of columns")
		}
		for i, item := range row {
			values[i] = item
		}
		out += fmt.Sprintf(table.Format, values...)
	}
	return out
}
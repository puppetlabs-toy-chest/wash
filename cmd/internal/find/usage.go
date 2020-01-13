package find

import (
	"github.com/puppetlabs/wash/cmd/internal/config"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// Usage returns `wash find`'s usage string
func Usage() string {
	use := "wash find"
	if config.Embedded {
		use = "find"
	}
	u := ""
	u += "NOTE: https://puppetlabs.github.io/wash/tutorials/02_find/ contains a hands-on\n"
	u += "tutorial of 'find'.\n"
	u += "\n"
	u += "Recursively descends the directory tree of the specified paths, evaluating an\n"
	u += "'expression' composed of 'primaries' and 'operands' for each entry in the tree.\n"
	u += "\n"
	u += "Usage:\n"
	u += "  " + use + " [paths] [options] [expression]\n"
	u += "\n"

	t := types.OptionsTable()
	addEmptyRow := func() {
		t.Append([][]string{
			[]string{"", ""},
		})
	}
	addEmptyRow()
	t.Append(primary.Table().Rows())
	addEmptyRow()
	t.Append(parser.OperandsTable().Rows())
	u += t.Format()
	u += "\n"

	u += "Use \"wash find --help <primary>\" for more information about a primary. To view\n"
	u += "a detailed description of find's expression syntax, use \"wash find --help syntax\".\n"
	u += "\n"
	u += "NOTE: The default reference time for all time predicates is find's start time.\n"
	u += "\n"
	u += "NOTE: All entry attribute primaries return false if the given entry does not have\n"
	u += "the specified attribute. For example, the -mtime primary will always return false\n"
	u += "if the entry does not have an mtime attribute.\n"
	u += "\n"
	u += "NOTE: find exits with status 0 if all entries are processed successfully, greater\n"
	u += "than 0 if errors occur. This is deliberately a very broad description, but if the\n"
	u += "return value is non-zero, you should not rely on the correctness of find.\n"

	return u
}

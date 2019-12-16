package cmdutil

import (
	"fmt"
)

// InputParser represents a parser that parses input
// passed into Prompt.
type InputParser = func(string) (interface{}, error)

// YesOrNoP parses input representing confirmation. confirmed (bool) is
// true if the input starts with "y" or "Y".
var YesOrNoP InputParser = func(input string) (confirmed interface{}, err error) {
	confirmed = len(input) > 0 && (input[0] == 'y' || input[0] == 'Y')
	return
}

// Prompt prints the supplied message, waits for input on stdin,
// then passes the input over to the supplied parser. The actual
// prompt displayed to the user is "{msg} ".
func Prompt(msg string, parser InputParser) (interface{}, error) {
	stderrMux.Lock()
	defer stderrMux.Unlock()

	var input string
	fmt.Fprintf(Stderr, "%s ", msg)
	_, err := fmt.Scanln(&input)
	if err != nil {
		return nil, err
	}
	return parser(input)
}

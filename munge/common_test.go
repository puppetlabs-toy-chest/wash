package munge

import (
	"fmt"
	"regexp"

	"github.com/stretchr/testify/suite"
)

// This file stores some common test setup that's shared by all the
// munge functions.

type testCase struct {
	input    interface{}
	expected interface{}
	errRegex *regexp.Regexp
}

// nTC => newTestCase. It's meant to save some typing.
func nTC(input interface{}, v interface{}) testCase {
	tc := testCase{input: input}
	tc.expected = v
	return tc
}

// nETC => newErrorTestCase. It's meant to save some typing.
func nETC(input interface{}, errRegex string) testCase {
	tc := testCase{input: input}
	tc.errRegex = regexp.MustCompile(errRegex)
	return tc
}

type MungeTestSuite struct {
	suite.Suite
	// This should be set in each test.
	mungeFunc func(interface{}) (interface{}, error)
}

func (suite *MungeTestSuite) runTestCases(cases ...testCase) {
	var input interface{}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panicked on input %t\n", input)
			panic(r)
		}
	}()
	for _, c := range cases {
		input = c.input
		actual, err := suite.mungeFunc(input)
		if c.errRegex != nil {
			suite.Regexp(c.errRegex, err, "Input was %t", input)
		} else {
			if suite.NoError(err, "Input was %t", input) {
				suite.Equal(c.expected, actual, "Input was %t", input)
			}
		}
	}
}

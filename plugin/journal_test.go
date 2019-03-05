package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

type testJournal struct {
	mock.Mock
}

func (j *testJournal) Log(msg string, a ...interface{}) {
	j.Called(msg, a)
}

func TestLog(t *testing.T) {
	var j testJournal
	msg := "hello"
	var expected []interface{}
	j.On("Log", msg, expected)

	Log(context.WithValue(context.Background(), Journal, &j), msg)
	j.AssertExpectations(t)
}

func TestLogWithArgs(t *testing.T) {
	var j testJournal
	msg, arg := "hello %v", "world"
	j.On("Log", msg, []interface{}{arg})

	Log(context.WithValue(context.Background(), Journal, &j), msg, arg)
	j.AssertExpectations(t)
}

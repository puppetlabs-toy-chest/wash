package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

type testJournal struct {
	mock.Mock
}

func (j *testJournal) Log(msg string) {
	j.Called(msg)
}

func TestLog(t *testing.T) {
	var j testJournal
	msg := "hello"
	j.On("Log", msg)

	Log(context.WithValue(context.Background(), Journal, &j), msg)

	j.AssertExpectations(t)
}

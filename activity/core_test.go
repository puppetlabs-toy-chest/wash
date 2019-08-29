package activity

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRecord(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer CloseAll()

	// Log to a journal
	Record(context.WithValue(context.Background(), JournalKey, Journal{ID: "1"}), "hello there")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "1.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), "hello there")
	}
}

func TestLogExpired(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer CloseAll()

	// Ensure entries use a very short
	expires = 1 * time.Millisecond
	ctx := context.WithValue(context.Background(), JournalKey, Journal{ID: "2"})

	// Log twice, second after cache entry has expired
	Record(ctx, "first write")
	time.Sleep(1 * time.Millisecond)
	Record(ctx, "second write")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "2.log"))
	if assert.Nil(t, err) {
		assert.Regexp(t, "(?s)first write.*second write", string(bits))
	}
}

func TestLogReused(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer CloseAll()
	ctx := context.WithValue(context.Background(), JournalKey, Journal{ID: "3"})

	// Log twice
	Record(ctx, "first write")
	Record(ctx, "second %v", "write")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "3.log"))
	if assert.Nil(t, err) {
		assert.Regexp(t, "(?s)first write.*second write", string(bits))
	}
}

func TestDeadLetterOffice(t *testing.T) {
	Record(context.WithValue(context.Background(), JournalKey, Journal{}), "hello %v", "world")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "dead-letter-office.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), "hello world")
	}
}

func TestLogging(t *testing.T) {
	Record(context.Background(), "nobody home")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "dead-letter-office.log"))
	// Could get an error if dead-letter-office does not exist.
	if err == nil {
		assert.NotContains(t, string(bits), "nobody home")
	}
}

func TestSubmitMethodInvocation_NewMethodInvocation_SubmitsToGA(t *testing.T) {
	// Setup the mocks
	ctx := context.Background()
	journal := Journal{
		ID: "foo",
	}
	ctx = context.WithValue(ctx, JournalKey, journal)
	analyticsClient := &mockAnalyticsClient{}
	analyticsClient.On("Event", "Invocation", "Method", analytics.Params{
		"Label":      "List",
		"Plugin":     "foo",
	}).Return(nil)
	ctx = context.WithValue(ctx, analytics.ClientKey, analyticsClient)

	// Perform the test
	SubmitMethodInvocation(ctx, "foo", "foo::file", "List")
	assert.True(t, journal.recorder().methodInvoked("foo::file", "List"))
	analyticsClient.AssertExpectations(t)
}

func TestSubmitMethodInvocation_SubmittedMethodInvocation_DoesNotSubmitToGA(t *testing.T) {
	// Setup the mocks
	ctx := context.Background()
	journal := Journal{
		// Use a different journal ID so that we get a different recorder
		ID: "bar",
	}
	journal.recorder().submitMethodInvocation("foo::file", "List", func() {})
	ctx = context.WithValue(ctx, JournalKey, journal)
	analyticsClient := &mockAnalyticsClient{}
	analyticsClient.On("Event", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx = context.WithValue(ctx, analytics.ClientKey, analyticsClient)

	// Perform the test
	SubmitMethodInvocation(ctx, "foo", "foo::file", "List")
	analyticsClient.AssertNotCalled(t, "Event", mock.Anything, mock.Anything, mock.Anything)
}

func TestMain(m *testing.M) {
	dir, err := ioutil.TempDir("", "journal_tests")
	if err != nil {
		panic(err)
	}
	SetDir(dir)

	exitcode := m.Run()

	errz.Log(os.RemoveAll(dir))
	os.Exit(exitcode)
}

type mockAnalyticsClient struct {
	mock.Mock
}

func (c *mockAnalyticsClient) Screenview(name string, params analytics.Params) error {
	args := c.Called(name, params)
	return args.Error(0)
}

func (c *mockAnalyticsClient) Event(category string, action string, params analytics.Params) error {
	args := c.Called(category, action, params)
	return args.Error(0)
}

func (c *mockAnalyticsClient) Flush() {
}

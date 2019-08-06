package analytics

import (
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	oldHTTPClient  httpClientI
	mockHTTPClient *mockHTTPClient
	c              *client
}

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) post(url string, contentType string, body io.Reader) (*http.Response, error) {
	args := m.Called(url, contentType, body)
	return args.Get(0).(*http.Response), args.Error(1)
}

func (s *ClientTestSuite) SetupTest() {
	s.oldHTTPClient = httpClient
	s.mockHTTPClient = &mockHTTPClient{}
	httpClient = s.mockHTTPClient
	config := Config{
		Disabled: false,
		UserID:   uuid.New(),
	}
	s.c = NewClient(config).(*client)
}

func (s *ClientTestSuite) TearDownTest() {
	httpClient = s.oldHTTPClient
}

func (s *ClientTestSuite) TestNewClient_AnalyticsDisabled_ReturnsNoopClient() {
	config := Config{
		Disabled: true,
		UserID:   uuid.New(),
	}
	c := NewClient(config)
	_, ok := c.(*noopClient)
	s.True(ok, "NewClient does not return a noopClient when analytics is disabled")
}

// No need for a corresponding "TestNewClient_AnalyticsEnabled_ReturnsClient" test
// case because SetupTest already tests that behavior

func (s *ClientTestSuite) TestScreenview_EmptyName_ReturnsError() {
	err := s.c.Screenview("", Params{})
	s.Regexp("name.*required", err)
}

func (s *ClientTestSuite) TestScreenview_InvalidCustomDimension_ReturnsError() {
	err := s.c.Screenview("foo", Params{"Invalid Custom Dimension": "Foo"})
	s.Regexp("Invalid.*Dimension.*settable", err)
}

func (s *ClientTestSuite) TestScreenview_ValidInput_EnqueuesHit() {
	err := s.c.Screenview("foo", Params{})
	if s.NoError(err) {
		s.assertHits(Params{
			"t":  "screenview",
			"cd": "foo",
		})
	}
}

func (s *ClientTestSuite) TestEvent_EmptyCategory_ReturnsError() {
	err := s.c.Event("", "", Params{})
	s.Regexp("category.*required", err)
}

func (s *ClientTestSuite) TestEvent_EmptyAction_ReturnsError() {
	err := s.c.Event("category", "", Params{})
	s.Regexp("action.*required", err)
}

func (s *ClientTestSuite) TestEvent_InvalidCustomDimension_ReturnsError() {
	err := s.c.Event("fooCategory", "fooAction", Params{
		"Invalid Custom Dimension": "Foo",
	})
	s.Regexp("Invalid.*Dimension.*settable", err)
}

func (s *ClientTestSuite) TestEvent_ValidInput_EnqueuesHit() {
	err := s.c.Event("Invocation", "Method", Params{
		"Plugin":     "aws",
		"Entry Type": "aws::ec2Instance",
	})
	if s.NoError(err) {
		s.assertHits(Params{
			"t":   "event",
			"ec":  "Invocation",
			"ea":  "Method",
			"cd2": "aws",
			"cd3": "aws::ec2Instance",
		})
	}
}

func (s *ClientTestSuite) TestEvent_ValidInput_WithLabel_EnqueuesHit() {
	err := s.c.Event("Invocation", "Method", Params{
		"Label":      "List",
		"Plugin":     "aws",
		"Entry Type": "aws::ec2Instance",
	})
	if s.NoError(err) {
		s.assertHits(Params{
			"t":   "event",
			"ec":  "Invocation",
			"ea":  "Method",
			"el":  "List",
			"cd2": "aws",
			"cd3": "aws::ec2Instance",
		})
	}
}

func (s *ClientTestSuite) TestEvent_ValidInput_WithValue_EnqueuesHit() {
	err := s.c.Event("Invocation", "Method", Params{
		"Value":      "27",
		"Plugin":     "aws",
		"Entry Type": "aws::ec2Instance",
	})
	if s.NoError(err) {
		s.assertHits(Params{
			"t":   "event",
			"ec":  "Invocation",
			"ea":  "Method",
			"ev":  "27",
			"cd2": "aws",
			"cd3": "aws::ec2Instance",
		})
	}
}

func (s *ClientTestSuite) TestFlush_NoQueuedHits() {
	s.c.Flush()
	s.mockHTTPClient.AssertNotCalled(s.T(), "post")
}

func (s *ClientTestSuite) TestFlush_QueuedHits() {
	// Enqueue some hits
	err := s.c.Screenview("foo", Params{})
	if err != nil {
		s.FailNowf("Received unexpected error: %v", err.Error())
	}
	err = s.c.Event("Invocation", "Method", Params{
		"Plugin":     "aws",
		"Entry Type": "aws::ec2Instance",
	})
	if err != nil {
		s.FailNowf("Received unexpected error: %v", err.Error())
	}
	err = s.c.Screenview("bar", Params{})
	if err != nil {
		s.FailNowf("Received unexpected error: %v", err.Error())
	}

	expectedHits := []Params{
		Params{
			"t":  "screenview",
			"cd": "foo",
		},
		Params{
			"t":   "event",
			"ec":  "Invocation",
			"ea":  "Method",
			"cd2": "aws",
			"cd3": "aws::ec2Instance",
		},
		Params{
			"t":  "screenview",
			"cd": "bar",
		},
	}
	if s.assertHits(expectedHits...) {
		// Setup the mocks
		baseParams := s.c.baseParams()
		var expectedPayload []string
		for _, hit := range expectedHits {
			expectedPayload = append(expectedPayload, hit.merge(baseParams).encode())
		}
		s.mockHTTPClient.On(
			"post",
			"https://www.google-analytics.com/batch",
			"application/x-www-form-urlencoded",
			mock.MatchedBy(s.makeIOReaderMatcher(strings.Join(expectedPayload, "\n"))),
		).Return(&http.Response{}, nil)

		// Do the test
		s.c.Flush()

		// Perform the assertions
		s.mockHTTPClient.AssertExpectations(s.T())
		s.assertHits()
	}
}

func (s *ClientTestSuite) TestEnqueue_FullQueue_SubmitsAnalytics() {
	// Fill-up the queue
	for i := 0; i < maxHits; i++ {
		if err := s.c.Screenview("foo", Params{}); err != nil {
			s.FailNowf("Received unexpected error: %v", err.Error())
		}
	}

	// Setup the mocks
	s.mockHTTPClient.On(
		"post",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(&http.Response{}, nil)

	// Attempt to send another hit
	if err := s.c.Screenview("bar", Params{}); err != nil {
		s.FailNowf("Received unexpected error: %v", err.Error())
	}

	// All previous hits in the queue should be flushed.
	// Only the newly submitted hit should remain
	s.mockHTTPClient.AssertCalled(
		s.T(),
		"post",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
	s.assertHits(Params{
		"t":  "screenview",
		"cd": "bar",
	})
}

func (s *ClientTestSuite) TestBaseParams() {
	baseParams := s.c.baseParams()

	// These base params' values should rarely change
	constantBaseParams := Params{
		"v":   "1",
		"cid": s.c.userID.String(),
		"tid": "UA-144659575-2",
		"an":  "Wash",
		"aip": "true",
		"cd1": runtime.GOARCH,
	}
	for param, value := range constantBaseParams {
		if s.Contains(baseParams, param) {
			s.Equal(value, baseParams[param])
		}
	}

	// These base params' values will change often
	variableBaseParams := []string{
		"v",
	}
	for _, param := range variableBaseParams {
		s.Contains(baseParams, param)
	}
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) assertHits(hits ...Params) bool {
	if !s.Equal(len(hits), len(s.c.queuedHits)) {
		return false
	}
	for i := range hits {
		if !s.Equal(hits[i], s.c.queuedHits[i]) {
			return false
		}
	}
	return true
}

type IOReaderMatcherFunc = func(io.Reader) bool

func (s *ClientTestSuite) makeIOReaderMatcher(expectedBody string) IOReaderMatcherFunc {
	return func(bodyRdr io.Reader) bool {
		body, err := ioutil.ReadAll(bodyRdr)
		if err != nil {
			s.FailNowf("Unpexected error reading the request body: %v", err.Error())
		}
		return string(body) == expectedBody
	}
}

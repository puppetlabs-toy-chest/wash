package analytics

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-attic/jibber_jabber"
	"github.com/google/uuid"
	"github.com/puppetlabs/wash/cmd/version"
	log "github.com/sirupsen/logrus"
)

// Client represents a Google Analytics client. Hits are submitted to
// GA in batches of 20 to avoid overloading the network.
//
// Screenview queues a screenview hit, while Event queues an event hit.
// Params represents additional measurement protocol parameters to
// pass into Client#Screenview or Client#Event. These will typically be
// custom dimension values. The currently supported custom dimensions
// are "Plugin" and "Entry Type". Note that Event hits can also specify
// the event's (optional) label and value via the "Label" and "Value"
// keys in params.
//
// NOTE: See https://developers.google.com/analytics/devguides/collection/protocol/v1/parameters
// for more details about the measurement protocol.
//
// NOTE: Screenview and Event will only return an error for invalid input.
// If you always expect valid input, then feel free to ignore the error or
// to panic on it.
type Client interface {
	Screenview(name string, params Params) error
	Event(category string, action string, params Params) error
	Flush()
}

// NewClient returns a new Google Analytics client for submitting Wash analytics.
func NewClient(config Config) Client {
	if config.Disabled {
		log.Debugf("Analytics opt-out is set, analytics will be disabled")
		return &noopClient{}
	}
	client := &client{
		userID: config.UserID,
	}
	// Periodically flush queued analytics hits
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			<-ticker.C
			client.Flush()
		}
	}()
	return client
}

const maxHits = 20

type client struct {
	userID     uuid.UUID
	queueLock  sync.Mutex
	queuedHits []Params
}

func (c *client) Screenview(name string, params Params) error {
	s, err := newScreenview(name, params)
	if err != nil {
		return err
	}
	log.Debugf("%v received", s)
	c.enqueue(s.ToHit())
	return nil
}

func (c *client) Event(category string, action string, params Params) error {
	e, err := newEvent(category, action, params)
	if err != nil {
		return err
	}
	log.Debugf("%v received", e)
	c.enqueue(e.ToHit())
	return nil
}

func (c *client) Flush() {
	c.queueLock.Lock()
	defer c.queueLock.Unlock()
	c.flush()
}

func (c *client) enqueue(hit Params) {
	c.queueLock.Lock()
	defer c.queueLock.Unlock()
	if len(c.queuedHits) >= maxHits {
		c.flush()
	}
	c.queuedHits = append(c.queuedHits, hit)
}

func (c *client) flush() {
	if len(c.queuedHits) <= 0 {
		return
	}
	// According to https://developers.google.com/analytics/devguides/collection/protocol/v1/devguide#batch,
	// each line in the batch request's body represents a single hit. Thus, we log
	// something like
	//     Submitting analytics... (<base_params>)
	//     Payload:
	//       <first_hit>
	//       ...
	//       <last_hit>
	// before submitting the data.
	baseParams := c.baseParams()
	var logMsg strings.Builder
	logMsg.WriteString(fmt.Sprintf("Submitting analytics... (%v)\nPayload:\n", baseParams))
	var payload []string
	for _, hit := range c.queuedHits {
		payload = append(payload, hit.merge(baseParams).encode())
		logMsg.WriteString(fmt.Sprintf("  %v\n", hit))
	}
	log.Debug(logMsg.String())
	body := strings.Join(payload, "\n")
	// The Measurement Protocol's docs indicate that the endpoint
	// will always return a 200 OK status, even if the request
	// contains any errors. Thus, the response is useless so it is
	// OK for us to ignore it.
	_, err := httpClient.post(
		"https://www.google-analytics.com/batch",
		"application/x-www-form-urlencoded",
		strings.NewReader(body),
	)
	if err != nil {
		log.Infof("Failed to send analytics: %v", err)
	}
	c.queuedHits = c.queuedHits[:0]
}

func (c *client) baseParams() Params {
	params := Params{
		// v => Protocol Version
		"v": "1",
		// cid => Client ID
		"cid": c.userID.String(),
		// tid => Tracking ID
		"tid": "UA-144580607-1",
		// an => Application Name
		"an": "Wash",
		// av => Application Version
		"av": version.BuildVersion,
		// aip => Anonymize IPs
		"aip": "true",
		// cd1 => Custom Dimension 1 (Architecture)
		"cd1": runtime.GOARCH,
	}
	locale, err := jibber_jabber.DetectIETF()
	if err != nil {
		log.Infof("(Google Analytics) Could not detect the user locale: %v", err)
	} else {
		// ul => User Locale
		params["ul"] = locale
	}
	return params
}

var settableCustomDimensions = map[string]string{
	"Plugin": "cd2",
}

func mungeCustomDimensions(customDimensions Params) (Params, error) {
	mungedCustomDimensions := Params{}
	for cd, value := range customDimensions {
		index, ok := settableCustomDimensions[cd]
		if !ok {
			var settableCustomDimensionsArray []string
			for settableCD := range settableCustomDimensions {
				settableCustomDimensionsArray = append(settableCustomDimensionsArray, settableCD)
			}
			return nil, fmt.Errorf(
				"%v is not a settable custom dimension. Settable custom dimensions are: %v",
				cd,
				strings.Join(settableCustomDimensionsArray, ","),
			)
		}
		delete(customDimensions, cd)
		mungedCustomDimensions[index] = value
	}
	return mungedCustomDimensions, nil
}

type event struct {
	category string
	action   string
	label    string
	value    string
	params   Params
}

func newEvent(category string, action string, params Params) (event, error) {
	e := event{}

	if len(category) <= 0 {
		return e, fmt.Errorf("the event category is required")
	}
	if len(action) <= 0 {
		return e, fmt.Errorf("the event action is required")
	}
	e.category = category
	e.action = action

	e.label = params.delete("Label")
	e.value = params.delete("Value")
	e.params = params

	params, err := mungeCustomDimensions(params)
	if err != nil {
		return e, fmt.Errorf("%v submission error: %v", e, err)
	}
	e.params = params

	return e, nil
}

func (e event) ToHit() Params {
	params := Params{}
	// t => Hit Type
	params["t"] = "event"
	// ec => Event Category
	params["ec"] = e.category
	// ea => Event Action
	params["ea"] = e.action
	if e.label != "" {
		// el => Event Label
		params["el"] = e.label
	}
	if e.value != "" {
		// ev => Event Value
		params["ev"] = e.value
	}
	return params.merge(e.params)
}

func (e event) String() string {
	var eventStr strings.Builder
	eventStr.WriteString(fmt.Sprintf("'%v %v", e.category, e.action))
	if e.label != "" {
		eventStr.WriteString(" ")
		eventStr.WriteString(e.label)
	}
	if e.value != "" {
		eventStr.WriteString(" ")
		eventStr.WriteString(e.value)
	}
	eventStr.WriteString(fmt.Sprintf("' event (%v)", e.params))
	return eventStr.String()
}

type screenview struct {
	name   string
	params Params
}

func newScreenview(name string, params Params) (screenview, error) {
	s := screenview{}

	if len(name) <= 0 {
		return s, fmt.Errorf("the screen name is required")
	}
	s.name = name
	s.params = params

	params, err := mungeCustomDimensions(params)
	if err != nil {
		return s, fmt.Errorf(
			"%v submission error: %v",
			s,
			err,
		)
	}
	s.params = params

	return s, nil
}

func (s screenview) ToHit() Params {
	params := Params{}
	// t => Hit Type
	params["t"] = "screenview"
	// cd => Screen Name
	params["cd"] = s.name
	return s.params.merge(params)
}

func (s screenview) String() string {
	return fmt.Sprintf("'%v' screenview (%v)", s.name, s.params)
}

// The code below makes it possible for the tests to mock the
// HTTP client

type httpClientI interface {
	post(string, string, io.Reader) (*http.Response, error)
}

type httpClientImpl struct{}

func (httpClientImpl) post(url string, contentType string, body io.Reader) (*http.Response, error) {
	return http.Post(url, contentType, body)
}

var httpClient httpClientI = httpClientImpl{}

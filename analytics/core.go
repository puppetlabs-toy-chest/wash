// Package analytics provides tools for sending over Wash events and screenviews
// to Google Analytics
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// KeyType is used to type keys for looking up context values.
type KeyType int

// ClientKey is used to identify an analytics client in a context.
const ClientKey KeyType = iota

// FlushDuration represents the amount of time of a typical Client#Flush
// operation. Users of the analytics package should use this value when
// they need to flush outstanding analytics hits prior to exiting their
// application.
const FlushDuration = 250 * time.Millisecond

// GetClient retrieves the analytics client from the provided
// context. If the context does not contain an analytics client,
// then this will return a silenced noopClient. The latter's
// useful for testing.
func GetClient(ctx context.Context) Client {
	client := ctx.Value(ClientKey)
	if client != nil {
		return client.(Client)
	}
	return &noopClient{
		silenced: true,
	}
}

// Params represents additional measurement protocol parameters to
// pass into Client#Screenview or Client#Event.
type Params map[string]string

func (p Params) String() string {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		msg := fmt.Sprintf("Unexpected error when marshalling an analytics.Params object: %v", err)
		panic(msg)
	}
	return string(jsonBytes)
}

func (p Params) delete(key string) string {
	value := p[key]
	delete(p, key)
	return value
}

func (p Params) encode() string {
	urlValues := url.Values{}
	for param, value := range p {
		urlValues.Add(param, value)
	}
	return urlValues.Encode()
}

func (p Params) merge(p2 Params) Params {
	for param, value := range p2 {
		p[param] = value
	}
	return p
}

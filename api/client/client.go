// Package client provides helpers for interacting with the wash socket HTTP API.
package client

// This package is named `client` instead of `apiclient` because a client
// implies an API, so it is a bit redundant. However, because a client implies
// an API, it makes sense to include this code in an api/client/ directory.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/analytics"
	apitypes "github.com/puppetlabs/wash/api/types"
)

// Client represents a Wash API client.
type Client interface {
	Info(path string) (apitypes.Entry, error)
	List(path string) ([]apitypes.Entry, error)
	Metadata(path string) (map[string]interface{}, error)
	Stream(path string) (io.ReadCloser, error)
	Exec(path string, command string, args []string, opts apitypes.ExecOptions) (<-chan apitypes.ExecPacket, error)
	History(bool) (chan apitypes.Activity, error)
	ActivityJournal(index int, follow bool) (io.ReadCloser, error)
	Clear(path string) ([]string, error)
	// A "nil" schema means that the schema's unknown.
	Schema(path string) (*apitypes.EntrySchema, error)
	Screenview(name string, params analytics.Params) error
	Delete(path string) (bool, error)
	Signal(path string, signal string) error
}

// A domainSocketClient is a wash API client.
type domainSocketClient struct {
	*http.Client
}

var domainSocketBaseURL = "http://localhost"

// ForUNIXSocket returns a client suitable for making wash API calls over a UNIX
// domain socket.
func ForUNIXSocket(pathToSocket string) Client {
	return &domainSocketClient{
		&http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", pathToSocket)
				},
			},
		}}
}

func unmarshalErrorResp(resp *http.Response) error {
	var errorObj apitypes.ErrorObj
	respBody, err := ioutil.ReadAll(resp.Body)
	errz.Log(resp.Body.Close())
	if err != nil {
		return err
	}
	if err := json.Unmarshal(respBody, &errorObj); err != nil {
		return fmt.Errorf("Not-OK status: %v, URL: %v, Body: %v", resp.StatusCode, resp.Request.URL.Path, string(respBody))
	}
	return &errorObj
}

func (c *domainSocketClient) doRequest(method, endpoint string, params url.Values, body io.Reader) (io.ReadCloser, error) {
	// Do common parameter munging.
	if paths, ok := params["path"]; ok {
		if len(paths) != 1 {
			panic("path parameter should have a single element")
		}
		path, err := filepath.Abs(paths[0])
		if err != nil {
			return nil, fmt.Errorf("could not calculate the absolute path of %v: %v", path, err)
		}
		params["path"] = []string{path}
	}

	req, err := http.NewRequest(method, domainSocketBaseURL, body)
	if err != nil {
		return nil, err
	}

	req.URL.Path = endpoint
	req.URL.RawQuery = params.Encode()

	journal := activity.JournalForPID(os.Getpid())
	req.Header.Set(apitypes.JournalIDHeader, journal.ID)
	req.Header.Set(apitypes.JournalDescHeader, journal.Description)
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return resp.Body, nil
	}

	return nil, unmarshalErrorResp(resp)
}

func (c *domainSocketClient) doRequestAndParseJSONBody(method, endpoint string, params url.Values, body io.Reader, result interface{}) error {
	respBody, err := c.doRequest(method, endpoint, params, body)
	if err != nil {
		return err
	}

	defer func() { errz.Log(respBody.Close()) }()
	respBodyBytes, err := ioutil.ReadAll(respBody)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(respBodyBytes, result); err != nil {
		return fmt.Errorf("Non-JSON body at %v: %v\n%v", endpoint, err, string(respBodyBytes))
	}

	return nil
}

func (c *domainSocketClient) getRequest(endpoint string, params url.Values, result interface{}) error {
	return c.doRequestAndParseJSONBody(http.MethodGet, endpoint, params, nil, result)
}

// Info retrieves the information of the resource located at "path"
func (c *domainSocketClient) Info(path string) (apitypes.Entry, error) {
	var e apitypes.Entry
	if err := c.getRequest("/fs/info", url.Values{"path": []string{path}}, &e); err != nil {
		return e, err
	}

	return e, nil
}

// List lists the resources located at "path".
func (c *domainSocketClient) List(path string) ([]apitypes.Entry, error) {
	var ls []apitypes.Entry
	if err := c.getRequest("/fs/list", url.Values{"path": []string{path}}, &ls); err != nil {
		return nil, err
	}

	return ls, nil
}

// Metadata gets the metadata of the resource located at "path".
func (c *domainSocketClient) Metadata(path string) (map[string]interface{}, error) {
	var metadata map[string]interface{}
	if err := c.getRequest("/fs/metadata", url.Values{"path": []string{path}}, &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// Stream updates for the resource located at "path".
func (c *domainSocketClient) Stream(path string) (io.ReadCloser, error) {
	respBody, err := c.doRequest(http.MethodGet, "/fs/stream", url.Values{"path": []string{path}}, nil)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

// Exec invokes the given command + args on the resource located at "path".
//
// The resulting channel contains events, ordered as we receive them from the
// server. The channel will be closed when there are no more events.
func (c *domainSocketClient) Exec(path string, command string, args []string, opts apitypes.ExecOptions) (<-chan apitypes.ExecPacket, error) {
	payload := apitypes.ExecBody{Cmd: command, Args: args, Opts: opts}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	respBody, err := c.doRequest(http.MethodPost, "/fs/exec", url.Values{"path": []string{path}}, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	readJSONFromBody := func(rdr io.ReadCloser, ch chan<- apitypes.ExecPacket) {
		defer func() { errz.Log(rdr.Close()) }()
		decoder := json.NewDecoder(rdr)
		for {
			var pkt apitypes.ExecPacket
			if err := decoder.Decode(&pkt); err == io.EOF {
				close(ch)
				return
			} else if err != nil {
				log.Println(err)
				close(ch)
				return
			} else {
				ch <- pkt
			}
		}
	}

	events := make(chan apitypes.ExecPacket, 1)
	go readJSONFromBody(respBody, events)
	return events, nil
}

// History returns a command history channel for the current wash server session.
// If follow is false, it closes when all current activity has been delivered.
func (c *domainSocketClient) History(follow bool) (chan apitypes.Activity, error) {
	var params url.Values
	if follow {
		params = url.Values{"follow": []string{"true"}}
	}
	respBody, err := c.doRequest(http.MethodGet, "/history", params, nil)
	if err != nil {
		return nil, err
	}

	acts := make(chan apitypes.Activity)
	go func() {
		defer func() { errz.Log(respBody.Close()) }()
		dec := json.NewDecoder(respBody)
		for dec.More() {
			var act apitypes.Activity
			if err := dec.Decode(&act); err != nil {
				log.Println(err)
				close(acts)
				return
			}
			acts <- act
		}
		close(acts)
	}()

	return acts, nil
}

// ActivityJournal returns a reader for the journal associated with a particular command in history.
// If follow is true, it streams new updates instead of returning the whole journal.
func (c *domainSocketClient) ActivityJournal(index int, follow bool) (io.ReadCloser, error) {
	var params url.Values
	if follow {
		params = url.Values{"follow": []string{"true"}}
	}
	return c.doRequest(http.MethodGet, "/history/"+strconv.Itoa(index), params, nil)
}

// Clear the cache at "path".
func (c *domainSocketClient) Clear(path string) ([]string, error) {
	respBody, err := c.doRequest(http.MethodDelete, "/cache", url.Values{"path": []string{path}}, nil)
	if err != nil {
		return nil, err
	}

	defer func() { errz.Log(respBody.Close()) }()
	body, err := ioutil.ReadAll(respBody)
	if err != nil {
		return nil, err
	}

	var result []string
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("Non-JSON body at %v: %v\n%v", "/cache", err, string(body))
	}

	return result, nil
}

// Schema returns the entry's schema
func (c *domainSocketClient) Schema(path string) (*apitypes.EntrySchema, error) {
	var schema *apitypes.EntrySchema
	if err := c.getRequest("/fs/schema", url.Values{"path": []string{path}}, &schema); err != nil {
		return schema, err
	}
	return schema, nil
}

// Screenview submits a screenview to Google Analytics
func (c *domainSocketClient) Screenview(name string, params analytics.Params) error {
	payload := apitypes.ScreenviewBody{
		Name:   name,
		Params: params,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.doRequest(http.MethodPost, "/analytics/screenview", url.Values{}, bytes.NewReader(jsonBody))
	return err
}

// Delete deletes the entry at "path"
func (c *domainSocketClient) Delete(path string) (bool, error) {
	var deleted bool
	err := c.doRequestAndParseJSONBody(http.MethodDelete, "/fs/delete", url.Values{"path": []string{path}}, nil, &deleted)
	return deleted, err
}

// Signal sends the given signal to tne entry at "path"
func (c *domainSocketClient) Signal(path string, signal string) error {
	payload := apitypes.SignalBody{Signal: signal}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.doRequest(http.MethodPost, "/fs/signal", url.Values{"path": []string{path}}, bytes.NewReader(jsonBody))
	return err
}

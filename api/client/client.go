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
	apitypes "github.com/puppetlabs/wash/api/types"
)

// Client represents a Wash API client.
type Client interface {
	Info(path string) (apitypes.Entry, error)
	List(path string) ([]apitypes.Entry, error)
	Metadata(path string) (map[string]interface{}, error)
	Stream(path string) (io.ReadCloser, error)
	Exec(path string, command string, args []string, opts apitypes.ExecOptions) (<-chan apitypes.ExecPacket, error)
	History() ([]apitypes.Activity, error)
	ActivityJournal(index int) (io.ReadCloser, error)
	Clear(path string) ([]string, error)
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

func (c *domainSocketClient) doRequest(method, endpoint, path string, body io.Reader) (io.ReadCloser, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("could not calculate the absolute path of %v: %v", path, err)
	}

	req, err := http.NewRequest(method, domainSocketBaseURL, body)
	if err != nil {
		return nil, err
	}

	req.URL.Path = endpoint
	req.URL.RawQuery = url.Values{"path": []string{path}}.Encode()

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

func (c *domainSocketClient) getRequest(endpoint, path string, result interface{}) error {
	respBody, err := c.doRequest(http.MethodGet, endpoint, path, nil)
	if err != nil {
		return err
	}

	defer func() { errz.Log(respBody.Close()) }()
	body, err := ioutil.ReadAll(respBody)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("Non-JSON body at %v: %v", endpoint, string(body))
	}

	return nil
}

// Info retrieves the information of the resource located at "path"
func (c *domainSocketClient) Info(path string) (apitypes.Entry, error) {
	var e apitypes.Entry
	if err := c.getRequest("/fs/info", path, &e); err != nil {
		return e, err
	}

	return e, nil
}

// List lists the resources located at "path".
func (c *domainSocketClient) List(path string) ([]apitypes.Entry, error) {
	var ls []apitypes.Entry
	if err := c.getRequest("/fs/list", path, &ls); err != nil {
		return nil, err
	}

	return ls, nil
}

// Metadata gets the metadata of the resource located at "path".
func (c *domainSocketClient) Metadata(path string) (map[string]interface{}, error) {
	var metadata map[string]interface{}
	if err := c.getRequest("/fs/metadata", path, &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// Stream updates for the resource located at "path".
func (c *domainSocketClient) Stream(path string) (io.ReadCloser, error) {
	respBody, err := c.doRequest(http.MethodGet, "/fs/stream", path, nil)
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

	respBody, err := c.doRequest(http.MethodPost, "/fs/exec", path, bytes.NewReader(jsonBody))
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

// History returns the command history for the current wash server session.
func (c *domainSocketClient) History() ([]apitypes.Activity, error) {
	// Intentionally skip journaling activity associated with history because that would modify it.
	resp, err := c.Get(domainSocketBaseURL + "/history")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, unmarshalErrorResp(resp)
	}

	defer func() { errz.Log(resp.Body.Close()) }()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result apitypes.HistoryResponse
	if err := json.Unmarshal(body, &result.Activities); err != nil {
		return nil, fmt.Errorf("Non-JSON body at %v: %v", "/history", string(body))
	}

	return result.Activities, nil
}

// ActivityJournal returns a reader for the journal associated with a particular command in history.
func (c *domainSocketClient) ActivityJournal(index int) (io.ReadCloser, error) {
	// Intentionally skip journaling activity associated with history because that would modify it.
	resp, err := c.Get(domainSocketBaseURL + "/history/" + strconv.Itoa(index))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, unmarshalErrorResp(resp)
	}

	return resp.Body, nil
}

// Clear the cache at "path".
func (c *domainSocketClient) Clear(path string) ([]string, error) {
	respBody, err := c.doRequest(http.MethodDelete, "/cache", path, nil)
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
		return nil, fmt.Errorf("Non-JSON body at %v: %v", "/cache", string(body))
	}

	return result, nil
}

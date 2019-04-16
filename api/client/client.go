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

	"github.com/Benchkram/errz"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/activity"
)

// A DomainSocketClient is a wash API client.
type DomainSocketClient struct {
	http.Client
}

var domainSocketBaseURL = "http://localhost"

// ForUNIXSocket returns a client suitable for making wash API calls over a UNIX
// domain socket.
func ForUNIXSocket(pathToSocket string) DomainSocketClient {
	return DomainSocketClient{
		http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", pathToSocket)
				},
			},
		}}
}

func (c *DomainSocketClient) doRequest(method, endpoint, path string, body io.Reader) (io.ReadCloser, error) {
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

	req.Header.Set(apitypes.JournalIDHeader, activity.PIDToID(os.Getpid()))
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return resp.Body, nil
	}

	var errorObj apitypes.ErrorObj
	respBody, err := ioutil.ReadAll(resp.Body)
	errz.Log(resp.Body.Close())
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(respBody, &errorObj); err != nil {
		return nil, fmt.Errorf("Not-OK status: %v, URL: %v, Body: %v", resp.StatusCode, endpoint, string(respBody))
	}

	return nil, &errorObj
}

func (c *DomainSocketClient) getRequest(endpoint, path string, result interface{}) error {
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
func (c *DomainSocketClient) Info(path string) (apitypes.Entry, error) {
	var e apitypes.Entry
	if err := c.getRequest("/fs/info", path, &e); err != nil {
		return e, err
	}

	return e, nil
}

// List lists the resources located at "path".
func (c *DomainSocketClient) List(path string) ([]apitypes.Entry, error) {
	var ls []apitypes.Entry
	if err := c.getRequest("/fs/list", path, &ls); err != nil {
		return nil, err
	}

	return ls, nil
}

// Metadata gets the metadata of the resource located at "path".
func (c *DomainSocketClient) Metadata(path string) (map[string]interface{}, error) {
	var metadata map[string]interface{}
	if err := c.getRequest("/fs/metadata", path, &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// Stream updates for the resource located at "path".
func (c *DomainSocketClient) Stream(path string) (io.ReadCloser, error) {
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
func (c *DomainSocketClient) Exec(path string, command string, args []string, opts apitypes.ExecOptions) (<-chan apitypes.ExecPacket, error) {
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

// Clear the cache at "path".
func (c *DomainSocketClient) Clear(path string) ([]string, error) {
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

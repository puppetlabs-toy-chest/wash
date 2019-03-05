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

	"github.com/Benchkram/errz"
	apitypes "github.com/puppetlabs/wash/api/types"

	"github.com/pkg/xattr"
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

func (c *DomainSocketClient) performRequest(endpoint string, result interface{}) error {
	url := fmt.Sprintf("%s%s", domainSocketBaseURL, endpoint)
	response, err := c.Get(url)
	if err != nil {
		log.Println(err)
		return err
	}

	defer func() { errz.Log(response.Body.Close()) }()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return err
	}

	if response.StatusCode != http.StatusOK {
		var errorObj apitypes.ErrorObj
		if err := json.Unmarshal(body, &errorObj); err != nil {
			return fmt.Errorf("Not-OK status: %v, URL: %v, Body: %v", response.StatusCode, endpoint, string(body))
		}

		return &errorObj
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("Non-JSON body at %v: %v", endpoint, string(body))
	}

	return nil
}

// List lists the resources located at "path".
func (c *DomainSocketClient) List(path string) ([]apitypes.ListEntry, error) {
	endpoint := fmt.Sprintf("/fs/list%s", path)

	var ls []apitypes.ListEntry
	if err := c.performRequest(endpoint, &ls); err != nil {
		return nil, err
	}

	return ls, nil
}

// Metadata gets the metadata of the resource located at "path".
func (c *DomainSocketClient) Metadata(path string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/fs/metadata%s", path)

	var metadata map[string]interface{}
	if err := c.performRequest(endpoint, &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// Exec invokes the given command + args on the resource located at "path".
//
// The resulting channel contains events, ordered as we receive them from the
// server. The channel will be closed when there are no more events.
func (c *DomainSocketClient) Exec(path string, command string, args []string, opts apitypes.ExecOptions) (<-chan apitypes.ExecPacket, error) {
	endpoint := fmt.Sprintf("/fs/exec%s", path)
	url := fmt.Sprintf("%s%s", domainSocketBaseURL, endpoint)

	// TODO: Extract out the handling of HTTP POST + JSON streaming into their own,
	// utility functions.

	payload := apitypes.ExecBody{Cmd: command, Args: args, Opts: opts}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	response, err := c.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		var errorObj apitypes.ErrorObj
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &errorObj); err != nil {
			return nil, fmt.Errorf("Not-OK status: %v, URL: %v, Body: %v", response.StatusCode, endpoint, string(body))
		}

		return nil, &errorObj
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
	go readJSONFromBody(response.Body, events)
	return events, nil
}

// APIKeyFromPath will take a path to an object within the wash filesystem,
// and interrogate it to determine its path relative to the wash filesystem
// root. This is stored in the extended attributes of every file in the wash fs.
func APIKeyFromPath(fspath string) (string, error) {
	p, err := xattr.Get(fspath, "wash.id")
	if err != nil {
		return "", err
	}
	return string(p), nil
}

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/api"

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
		var errorObj api.ErrorObj
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
func (c *DomainSocketClient) List(path string) ([]api.ListEntry, error) {
	endpoint := fmt.Sprintf("/fs/list%s", path)

	var ls []api.ListEntry
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

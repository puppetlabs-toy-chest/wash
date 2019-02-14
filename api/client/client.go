package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/pkg/xattr"
)

// A DomainSocketClient is a wash API client.
type DomainSocketClient struct {
	http.Client
}

var domainSocketBaseURL = "http://localhost"

// LSItem represents a single entry from the result of issuing a wash "list"
// request.
type LSItem struct {
	Commands   []string `json:"commands"`
	Name       string   `json:"name"`
	Attributes struct {
		Atime string `json:"Atime"`
		Mtime string `json:"Mtime"`
		Ctime string `json:"Ctime"`
		Mode  int    `json:"Mode"`
		Size  int    `json:"Size"`
		Valid int    `json:"Valid"`
	} `json:"attributes"`
}

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

func (c *DomainSocketClient) callResponse(path string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", domainSocketBaseURL, path)
	return c.Get(url)
}

// List lists the resources located at "path".
func (c *DomainSocketClient) List(path string) ([]LSItem, error) {
	url := fmt.Sprintf("/fs/list%s", path)
	response, err := c.callResponse(url)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		// Generate a real error object for this
		log.Printf("Status: %v, Body: %v", response.StatusCode, string(body))
		return nil, fmt.Errorf("Not-OK status: %v, URL: %v, Body: %v", response.StatusCode, path, string(body))
	}

	var ls []LSItem
	if err := json.Unmarshal(body, &ls); err != nil {
		return nil, fmt.Errorf("Non-JSON body at %v: %v", path, string(body))
	}

	return ls, nil
}

// APIPathFromXattrs will take a path to an object within the wash filesystem,
// and interrogate it to determine its path relative to the wash filesystem
// root. This is stored in the extended attributes of every file in the wash fs.
func APIPathFromXattrs(fspath string) (string, error) {
	p, err := xattr.Get(fspath, "wash.id")
	if err != nil {
		return "", err
	}
	return string(p), nil
}

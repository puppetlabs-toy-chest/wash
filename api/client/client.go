package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

type DomainSocketClient struct {
	http.Client
}

var domainSocketBaseURL = "http://localhost"

// TODO: consider moving this into the api package
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

func ClientUNIXSocket(pathToSocket string) DomainSocketClient {
	return DomainSocketClient{
		http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", pathToSocket)
				},
			},
		}}
}

func CallResponse(client DomainSocketClient, path string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", domainSocketBaseURL, path)
	return client.Get(url)
}

func List(client DomainSocketClient, path string) ([]LSItem, error) {
	url := fmt.Sprintf("/fs/list%s", path)
	response, err := CallResponse(client, url)
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

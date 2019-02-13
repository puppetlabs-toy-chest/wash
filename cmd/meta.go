package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/pkg/xattr"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
)

func metaCommand() *cobra.Command {
	metaCmd := &cobra.Command{
		Use:   "meta <file>",
		Short: "Prints the metadata of a file",
		Args:  cobra.MinimumNArgs(1),
	}

	metaCmd.Run = metaMain

	return metaCmd
}

func metaMain(cmd *cobra.Command, args []string) {
	path := args[0]
	socket := config.Fields.Socket

	apiPath, err := xattr.Get(path, "wash.id")
	if err != nil {
		// log.Fatal will exit the program with an exit code of 1
		log.Fatal(err)
	}

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
	}

	url := fmt.Sprintf("http://localhost/fs/metadata%v", string(apiPath))
	response, err := httpc.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		log.Fatal(fmt.Sprintf("Status: %v, Body: %v", response.StatusCode, string(body)))
	}

	var metadataBuffer bytes.Buffer
	json.Indent(&metadataBuffer, body, "", "  ")

	metadataBuffer.WriteTo(os.Stdout)

	os.Exit(0)
}

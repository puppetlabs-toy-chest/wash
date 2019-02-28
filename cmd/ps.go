package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/config"
)

func psCommand() *cobra.Command {
	psCmd := &cobra.Command{
		Use:   "ps [file...]",
		Short: "Lists the processes running on the indicated compute instances.",
	}

	psCmd.RunE = toRunE(psMain)

	return psCmd
}

func psMain(cmd *cobra.Command, args []string) exitCode {
	var paths []string
	if len(args) > 0 {
		paths = args
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return exitCode{1}
		}

		paths = []string{cwd}
	}

	var keys []string
	for _, path := range paths {
		apiKey, err := client.APIKeyFromPath(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to get API key for %v: %v\n", path, err)
		} else {
			keys = append(keys, apiKey)
		}
	}
	if len(keys) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no valid resources found")
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(config.Socket)

	// TODO: make this structured ps data.
	results := make(chan string, len(keys))
	for i, key := range keys {
		go func(k string, idx int) {
			ch, err := conn.Exec(k, "ps", []string{"-A"})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				results <- ""
				return
			}

			exit := 0
			result := k + "\n----\n"
			for pkt := range ch {
				switch pktType := pkt.TypeField; pktType {
				case api.Exitcode:
					exit = int(pkt.Data.(float64))
				case api.Stdout:
					result += pkt.Data.(string)
				case api.Stderr:
					fmt.Fprint(os.Stderr, pkt.Data)
				}
			}
			results <- result

			if exit != 0 {
				fmt.Fprintf(os.Stderr, "ps on %v exited %v\n", k, exit)
			}
		}(key, i)
	}

	first := true
	for range keys {
		if !first {
			fmt.Println()
		}
		first = false
		fmt.Print(<-results)
	}

	return exitCode{0}
}

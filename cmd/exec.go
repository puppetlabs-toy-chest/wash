package cmd

import (
	"fmt"
	"os"

	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
)

func execCommand() *cobra.Command {
	execCmd := &cobra.Command{
		Use:   "exec path command arg arg arg...",
		Short: "Executes the given command on the indicated target",
		Args:  cobra.MinimumNArgs(2),
	}

	execCmd.RunE = toRunE(execMain)

	// Don't interpret any flags after the first positional argument. Those should
	// instead get interpreted by this command as normal args, not flags.
	execCmd.Flags().SetInterspersed(false)

	return execCmd
}

func printPackets(pkts <-chan api.ExecPacket) int {
	exit := 0
	for pkt := range pkts {
		switch pktType := pkt.TypeField; pktType {
		case api.Exitcode:
			exit = int(pkt.Data.(float64))
		case api.Stdout:
			fmt.Print(pkt.Data)
		case api.Stderr:
			fmt.Fprint(os.Stderr, pkt.Data)
		}
	}

	return exit
}

func execMain(cmd *cobra.Command, args []string) exitCode {
	var path string
	var command string
	var commandArgs []string

	path = args[0]
	command = args[1]
	commandArgs = args[2:]

	apiPath, err := client.APIKeyFromPath(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(config.Socket)

	ch, err := conn.Exec(apiPath, command, commandArgs, api.ExecOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode{1}
	}

	code := printPackets(ch)
	return exitCode{code}
}

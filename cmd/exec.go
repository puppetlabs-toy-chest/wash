package cmd

import (
	"fmt"
	"os"

	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
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

func printPackets(pkts <-chan apitypes.ExecPacket) (int, error) {
	exit := 0
	foundErroredPacket := false

	for pkt := range pkts {
		if pkt.Err != nil {
			if !foundErroredPacket {
				// This is the first error we've encountered
				cmdutil.ErrPrintf("The exec endpoint errored. All incoming data will be ignored, with only the errors printed.\n")
				foundErroredPacket = true
			}

			cmdutil.ErrPrintf("%v\n", pkt.Err)
		}

		if foundErroredPacket {
			// This case handles the (unlikely) possibility that one of stdout/stderr
			// errors, while the other continues to send data. In that case, we want
			// to ignore the sent data.
			continue
		}

		switch pktType := pkt.TypeField; pktType {
		case apitypes.Exitcode:
			exit = int(pkt.Data.(float64))
		case apitypes.Stdout:
			fmt.Print(pkt.Data)
		case apitypes.Stderr:
			fmt.Fprint(os.Stderr, pkt.Data)
		}
	}

	if foundErroredPacket {
		return 0, fmt.Errorf("the exec endpoint errored")
	}

	return exit, nil
}

func execMain(cmd *cobra.Command, args []string) exitCode {
	var path string
	var command string
	var commandArgs []string

	path = args[0]
	command = args[1]
	commandArgs = args[2:]

	conn := client.ForUNIXSocket(config.Socket)

	ch, err := conn.Exec(path, command, commandArgs, apitypes.ExecOptions{})
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	code, err := printPackets(ch)
	if err != nil {
		return exitCode{1}
	}

	return exitCode{code}
}

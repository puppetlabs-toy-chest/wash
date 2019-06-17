package cmd

import (
	"fmt"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/cmd/internal/config"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func execCommand() *cobra.Command {
	use, aliases := "exec", []string{"wexec"}
	if config.Embedded {
		use, aliases = "wexec", []string{}
	}
	execCmd := &cobra.Command{
		Use:     use + " <path> <command> [<arg>...]",
		Aliases: aliases,
		Short:   "Executes the given command on the indicated target",
		Long: `For a Wash resource (specified by <path>) that implements the ability to execute a command, run the
specified command and arguments. The results will be forwarded from the target on stdout, stderr,
and exit code.`,
		Example: `exec docker/containers/example_1 printenv USER
  print the USER environment variable from a Docker container instance`,
		Args: cobra.MinimumNArgs(2),
		RunE: toRunE(execMain),
	}

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
			cmdutil.Print(pkt.Data)
		case apitypes.Stderr:
			fmt.Fprint(cmdutil.Stderr, pkt.Data)
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

	conn := cmdutil.NewClient()

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

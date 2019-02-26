package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
)

func execCommand() *cobra.Command {
	// TODO: Figure out how to stop cobra argument parsing past a point, to allow
	// for pass-through of args to the remote command.
	// e.g. "uname -a" should pass along the "-a", not freak out Cobra, which normally
	// assumes that the "-a" should apply to "wash exec"
	execCmd := &cobra.Command{
		Use:   "exec path command arg arg arg...",
		Short: "Executes the given command on the indicated target",
		Args:  cobra.MinimumNArgs(2),
	}

	execCmd.RunE = toRunE(execMain)

	return execCmd
}

var sigil = map[api.ExecPacketType]string{
	api.Stdout:   "out",
	api.Stderr:   "err",
	api.Exitcode: "wrn",
}

func formatOutputLine(event api.ExecPacket) string {
	format := `[%s] [%s] %s` + "\n"
	tstamp := event.Timestamp.Local().Format("15:04:05.00")
	sig := sigil[event.TypeField]

	if event.TypeField == api.Exitcode {
		line := fmt.Sprintf("Process exited with: %v", event.Data)
		return fmt.Sprintf(format, tstamp, sig, line)
	}

	line := fmt.Sprintf("%s", event.Data)
	line = strings.TrimSuffix(line, "\n")
	var output string
	for _, l := range strings.Split(line, "\n") {
		output += fmt.Sprintf(format, tstamp, sig, l)
	}

	return output
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

	ch, err := conn.Exec(apiPath, command, commandArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode{1}
	}

	for event := range ch {
		fmt.Print(formatOutputLine(event))
	}

	return exitCode{0}
}

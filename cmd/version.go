package cmd

import (
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/cmd/version"
	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print wash version",
		RunE:  toRunE(versionMain),
	}
	return versionCmd
}

func versionMain(cmd *cobra.Command, args []string) exitCode {
	cmdutil.Println(version.BuildVersion)
	return exitCode{0}
}

// Package cmd implements Wash's CLI using https://github.com/spf13/cobra.
package cmd

import (
	"github.com/puppetlabs/wash/cmd/internal/config"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

// Unfortunately, cobra.Command.Execute() can only return error objects.
// Thus, the only way for us to let each command configure its own exit
// code is to wrap that value in an error object. This should be OK since
// we want the commands to handle their own errors.
type exitCode struct {
	value int
}

// Required to implement the error interface
func (e exitCode) Error() string {
	return ""
}

// This munging's necessary to ensure that all commandMain functions return
// an exit code while also letting them be used as RunE functions that can
// be passed into Cobra. Otherwise, Go's type-checker will complain even though
// exitCode is an error object.
type commandMain func(cmd *cobra.Command, args []string) exitCode
type runE func(cmd *cobra.Command, args []string) error

func toRunE(main commandMain) runE {
	return func(cmd *cobra.Command, args []string) error {
		return main(cmd, args)
	}
}

func rootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:    "wash [<script>]",
		PreRun: bindServerArgs,
		RunE:   toRunE(rootMain),
		Long: `When invoked without arguments, enters a Wash shell. Starts the Wash daemon,
then starts your system shell with shortcuts configured for wash subcommands.`,
		// Need to set these so that Cobra will not output the usage +
		// error object when Execute() returns an error, which will always
		// happen in our case because the exitCode object is technically
		// an error.
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		Version:       version,
	}

	if config.Embedded {
		rootCmd.Use = ""
		rootCmd.PreRun = nil
		rootCmd.Long = ""
		// Augment the usage template to minimize usage when set to empty.
		rootCmd.SetUsageTemplate(embeddedUsageTemplate)
	} else {
		// Omit server from embedded cases because a daemon is already running.
		addServerArgs(rootCmd, "warn")
		rootCmd.AddCommand(serverCommand())
		// rootCommandFlag is used in rootMain.go.
		rootCmd.Flags().StringVarP(&rootCommandFlag, "command", "c", "", "Run the supplied string and exit")
	}

	rootCmd.AddCommand(versionCommand())
	rootCmd.AddCommand(metaCommand())
	rootCmd.AddCommand(listCommand())
	rootCmd.AddCommand(execCommand())
	rootCmd.AddCommand(psCommand())
	rootCmd.AddCommand(findCommand())
	rootCmd.AddCommand(clearCommand())
	rootCmd.AddCommand(tailCommand())
	rootCmd.AddCommand(historyCommand())
	rootCmd.AddCommand(infoCommand())
	rootCmd.AddCommand(streeCommand())

	return rootCmd
}

// Execute executes the root command, returning the exit code
func Execute() int {
	if err := config.Init(); err != nil {
		cmdutil.ErrPrintf("Failed to initialize Wash's config: %v", err)
		return 1
	}

	err := rootCommand().Execute()
	if err == nil {
		// This can happen if the user invokes `wash` without any
		// arguments, or if they invoke a help command.
		return 0
	}

	exitCode, ok := err.(exitCode)
	if !ok {
		// err is something Cobra-related, like e.g. a malformed
		// flag. Print the error, then return.
		cmdutil.ErrPrintf("Error: %v\n", err)
		return 1
	}

	return exitCode.value
}

// Usage template copied from Cobra and modified to format well with an empty Use clause.
const embeddedUsageTemplate = `Usage:{{if (and .Runnable (ne .Use ""))}}
 {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
 {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// Return use name and alias based on whether we're embedded in a wash shell.
func generateShellAlias(name string) (string, []string) {
	if config.Embedded {
		return "w" + name, []string{}
	}
	return name, []string{"w" + name}
}

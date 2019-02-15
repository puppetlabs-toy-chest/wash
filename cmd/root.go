package cmd

import (
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
		Use: "wash",
		// Need to set these so that Cobra will not output the usage +
		// error object when Execute() returns an error, which will always
		// happen in our case because the exitCode object is technically
		// an error.
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(serverCommand())
	rootCmd.AddCommand(metaCommand())
	rootCmd.AddCommand(lsCommand())

	return rootCmd
}

// Execute executes the root command, returning the exit code
func Execute() int {
	err := rootCommand().Execute()
	if err == nil {
		panic("The command did not return a valid exit code")
	}

	exitCode, ok := err.(exitCode)
	if !ok {
		panic("The command returned an error object instead of an exit code")
	}

	return exitCode.value
}

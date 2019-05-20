package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/api/client"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func historyCommand() *cobra.Command {
	historyCmd := &cobra.Command{
		Use:   "history [-f] [<id>]",
		Short: "Prints the wash command history, or journal of a particular item",
		Long: `Wash maintains a history of commands executed through it. Print that command history, or specify an
<id> to print a log of activity related to a particular command.`,
		Args: cobra.MaximumNArgs(1),
	}

	historyCmd.Flags().BoolP("follow", "f", false, "Follow new updates")
	if err := viper.BindPFlag("follow", historyCmd.Flags().Lookup("follow")); err != nil {
		cmdutil.ErrPrintf("%v\n", err)
	}

	historyCmd.RunE = toRunE(historyMain)

	return historyCmd
}

func printJournalEntry(index string, follow bool) error {
	idx, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	conn := client.ForUNIXSocket(config.Socket)
	// Translate from 1-indexing for history entries
	rdr, err := conn.ActivityJournal(idx-1, follow)
	if err != nil {
		return err
	}
	defer func() {
		errz.Log(rdr.Close())
	}()

	_, err = io.Copy(os.Stdout, rdr)
	return err
}

func printHistory(follow bool) error {
	conn := client.ForUNIXSocket(config.Socket)
	history, err := conn.History(follow)
	if err != nil {
		return err
	}

	// Use 1-indexing for history entries
	indexColumnLength := len(strconv.Itoa(len(history)))
	formatStr := "%" + strconv.Itoa(indexColumnLength) + "d  %s  %s\n"
	i := 0
	for item := range history {
		fmt.Printf(formatStr, i+1, item.Start.Format("2006-01-02 15:04"), item.Description)
		i++
	}
	return nil
}

func historyMain(cmd *cobra.Command, args []string) exitCode {
	follow := viper.GetBool("follow")

	var err error
	if len(args) > 0 {
		err = printJournalEntry(args[0], follow)
	} else {
		err = printHistory(follow)
	}

	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	return exitCode{0}
}

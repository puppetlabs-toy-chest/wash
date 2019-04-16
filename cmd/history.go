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
)

func historyCommand() *cobra.Command {
	historyCmd := &cobra.Command{
		Use:   "history [<id>]",
		Short: "Prints the wash command history, or journal of a particular item",
		Args:  cobra.MaximumNArgs(1),
	}

	historyCmd.RunE = toRunE(historyMain)

	return historyCmd
}

func printJournalEntry(index string) error {
	idx, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	conn := client.ForUNIXSocket(config.Socket)
	// Translate from 1-indexing for history entries
	rdr, err := conn.Journal(idx - 1)
	if err != nil {
		return err
	}
	defer func() {
		errz.Log(rdr.Close())
	}()

	_, err = io.Copy(os.Stdout, rdr)
	return err
}

func printHistory() error {
	conn := client.ForUNIXSocket(config.Socket)
	history, err := conn.History()
	if err != nil {
		return err
	}

	// Use 1-indexing for history entries
	indexColumnLength := len(strconv.Itoa(len(history)))
	formatStr := "%" + strconv.Itoa(indexColumnLength) + "d  %s  %s\n"
	for i, item := range history {
		fmt.Printf(formatStr, i+1, item.Start.Format("2006-01-02 15:04"), item.Description)
	}
	return nil
}

func historyMain(cmd *cobra.Command, args []string) exitCode {
	var err error
	if len(args) > 0 {
		err = printJournalEntry(args[0])
	} else {
		err = printHistory()
	}

	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	return exitCode{0}
}

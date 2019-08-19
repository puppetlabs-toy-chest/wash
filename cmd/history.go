package cmd

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Benchkram/errz"
	"github.com/kr/logfmt"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func historyCommand() *cobra.Command {
	use, aliases := generateShellAlias("history")
	historyCmd := &cobra.Command{
		Use:     use + " [-f] [<id>]",
		Aliases: aliases,
		Short:   "Prints the wash command history, or journal of a particular item",
		Long: `Wash maintains a history of commands executed through it. Print that command history, or specify an
<id> to print a log of activity related to a particular command.`,
		Args: cobra.MaximumNArgs(1),
		RunE: toRunE(historyMain),
	}
	historyCmd.Flags().BoolP("follow", "f", false, "Follow new updates")
	return historyCmd
}

type logFmtLine struct {
	Time, Level, Msg string
}

func printJournalEntry(index string, follow bool) error {
	idx, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	conn := cmdutil.NewClient()
	// Translate from 1-indexing for history entries
	rdr, err := conn.ActivityJournal(idx-1, follow)
	if err != nil {
		return err
	}
	defer func() {
		errz.Log(rdr.Close())
	}()

	// Output format:
	// Jun 13 15:44:04.299 Exec [find / -mindepth 1 -maxdepth 5 -exec stat -c %s %X %Y %Z %f %n {} +] on blissful_gould
	// Jun 13 15:44:04.433 stdout: 4096 1559079604 1557434981 1559079604 41ed /lib
	//                     2597536 1552660099 1552660099 1559079604 81ed /lib/libcrypto.so.1.1
	scanner := bufio.NewScanner(rdr)
	for scanner.Scan() {
		var line, empty logFmtLine
		if err := logfmt.Unmarshal(scanner.Bytes(), &line); err != nil {
			cmdutil.ErrPrintf("Error parsing %v: %v\n", line, err)
			continue
		}

		if line == empty {
			// Parser ignored incomplete line rather than erroring. Skip it.
			continue
		}

		// TODO: add option to print the original longer time format.
		t, err := time.Parse(time.RFC3339Nano, line.Time)
		if err != nil {
			panic(fmt.Sprintf("Unexpected time format %s", line.Time))
		}

		lines := strings.Split(line.Msg, "\n")
		timeStr := t.Format(time.StampMilli)
		fmt.Println(timeStr, lines[0])
		if len(lines) > 1 {
			prefix := strings.Repeat(" ", len(timeStr))
			for _, l := range lines[1:] {
				fmt.Println(prefix, l)
			}
		}
	}

	return nil
}

func printHistory(follow bool) error {
	conn := cmdutil.NewClient()
	history, err := conn.History(follow)
	if err != nil {
		return err
	}

	// Use 1-indexing for history entries
	indexColumnLength := len(strconv.Itoa(len(history)))
	formatStr := "%" + strconv.Itoa(indexColumnLength) + "d  %s  %s\n"
	i := 0
	for item := range history {
		cmdutil.Printf(formatStr, i+1, item.Start.Format("2006-01-02 15:04"), item.Description)
		i++
	}
	return nil
}

func historyMain(cmd *cobra.Command, args []string) exitCode {
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		panic(err.Error())
	}

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

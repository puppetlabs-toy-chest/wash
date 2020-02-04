package cmd

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
)

func psCommand() *cobra.Command {
	use, aliases := generateShellAlias("ps")
	psCmd := &cobra.Command{
		Use:     use + " [<node>...]",
		Aliases: aliases,
		Short:   "Lists the processes running on the indicated compute instances",
		Long: `Captures /proc/*/{cmdline,stat,statm} on each node by executing 'cat' on them. Collects the output
to display running processes on all listed nodes. Errors on paths that don't implement exec.`,
		RunE: toRunE(psMain),
	}
	return psCmd
}

func collectOutput(ch <-chan apitypes.ExecPacket) (string, error) {
	exit := 0
	var stdout, stderr string
	for pkt := range ch {
		if pkt.Err != nil {
			return "", pkt.Err
		}

		switch pktType := pkt.TypeField; pktType {
		case apitypes.Exitcode:
			exit = int(pkt.Data.(float64))
		case apitypes.Stdout:
			stdout += pkt.Data.(string)
		case apitypes.Stderr:
			stderr += pkt.Data.(string)
		}
	}

	if exit != 0 {
		return "", fmt.Errorf("ps exited %v\n%v", exit, stdout+stderr)
	}
	return stdout, nil
}

type psresult struct {
	pid     int
	active  time.Duration
	command string
}

func formatStats(paths []string, results map[string][]psresult) string {
	headers := []cmdutil.ColumnHeader{
		{ShortName: "node", FullName: "NODE"},
		{ShortName: "pid", FullName: "PID"},
		{ShortName: "time", FullName: "TIME"},
		{ShortName: "cmd", FullName: "COMMAND"},
	}
	var table [][]string
	for _, path := range paths {
		// Shorten path segments to probably-unique short strings, like `ku*s/do*p/de*t/pods/redis`.
		for _, st := range results[path] {
			segments := strings.Split(strings.Trim(path, "/"), "/")
			for i, segment := range segments[:len(segments)-1] {
				if len(segment) > 4 {
					segments[i] = segment[:2] + "*" + segment[len(segment)-1:]
				}
			}

			table = append(table, []string{
				strings.Join(segments, "/"),
				strconv.Itoa(st.pid),
				cmdutil.FormatDuration(st.active),
				st.command,
			})
		}
	}
	return cmdutil.NewTableWithHeaders(headers, table).Format()
}

func psMain(cmd *cobra.Command, args []string) exitCode {
	var paths []string
	if len(args) > 0 {
		paths = args
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}

		paths = []string{cwd}
	}

	conn := cmdutil.NewClient()

	results := make(map[string][]psresult)
	// Prepulate the map so it doesn't change size while all the goroutines are adding data.
	for _, path := range paths {
		results[path] = []psresult{}
	}

	var wg sync.WaitGroup
	wg.Add(len(paths))
	for i, path := range paths {
		go func(k string, idx int) {
			defer wg.Done()

			entry, err := conn.Info(k)
			if err != nil {
				cmdutil.ErrPrintf("errored on %v: %v\n", k, err)
			}
			var shell plugin.LoginShell
			if entry.Attributes.HasLoginShell() {
				shell = entry.Attributes.LoginShell()
			}
			if shell == plugin.UnknownShell {
				// Assume posix if unknown
				shell = plugin.POSIXShell
			}

			dispatcher := dispatchers[shell]
			ch, err := dispatcher.execPS(conn, k)
			if err != nil {
				cmdutil.ErrPrintf("errored on %v: %v\n", k, err)
				return
			}
			out, err := collectOutput(ch)
			if err != nil {
				cmdutil.ErrPrintf("errored on %v: %v\n", k, err)
				return
			}

			results[k], err = dispatcher.parseOutput(out)
			if err != nil {
				cmdutil.ErrPrintf("errored on %v: %v\n", k, err)
			}
		}(path, i)
	}

	wg.Wait()
	cmdutil.Print(formatStats(paths, results))
	return exitCode{0}
}

var dispatchers = []struct {
	execPS      func(client.Client, string) (<-chan apitypes.ExecPacket, error)
	parseOutput func(string) ([]psresult, error)
}{
	{}, // Unknown
	{ // POSIX shell
		execPS: func(conn client.Client, name string) (<-chan apitypes.ExecPacket, error) {
			return conn.Exec(name, "sh", []string{}, apitypes.ExecOptions{Input: psScript})
		},
		parseOutput: parseStatLines,
	},
	{ // PowerShell
		execPS: func(conn client.Client, name string) (<-chan apitypes.ExecPacket, error) {
			cmd := "Get-Process | Where TotalProcessorTime | Where Path | " +
				"Select-Object -Property Id,TotalProcessorTime,Path | ConvertTo-Csv"
			return conn.Exec(name, cmd, []string{}, apitypes.ExecOptions{})
		},
		parseOutput: parseCsvLines,
	},
}

// PS POSIX

// Get the list of processes separately from iterating over them. This avoids including 'find' as
// one of the active processes. Also exclude the pid of the shell we use to run this script. Uses
// printf to put everything on one line; \0-terminated cmdline, then stat, then \0 and statm.
// Proc parsing: http://man7.org/linux/man-pages/man5/proc.5.html
const psScript = `
procs=$(find /proc -maxdepth 1 -regex '.*/[0-9]*')
pid=$$
for i in $procs; do
	if [ -d $i -a ${i#/proc/} -ne $pid ]; then
	  cmdline=$(cat $i/cmdline | tr '\0' ' ')
		printf '%s\t%s\t%s\n' "$cmdline" "$(cat $i/stat)" "$(cat $i/statm)"
	fi
done
`

// Assume _SC_CLK_TCK is 100Hz for now. Can maybe get with 'getconf CLK_TCK'.
const clockTick = 100

func parseStatEntry(line string) (psresult, error) {
	tokens := strings.Split(line, "\t")
	if len(tokens) != 3 {
		return psresult{}, fmt.Errorf("Line had %v, not 3 tokens: %#v", len(tokens), tokens)
	}
	stat := strings.TrimSpace(tokens[1])
	// statm := tokens[2]
	command := strings.TrimSpace(tokens[0])

	var pid, ppid, pgrp, session, ttynr, tpgid int
	var flags uint
	var minflt, cminflt, majflt, cmajflt, utime, stime uint64
	var comm string
	var state rune
	statf := "%d %s %c %d %d %d %d %d %d %d %d %d %d %d %d"
	n, err := fmt.Sscanf(stat, statf,
		&pid, &comm, &state, &ppid, &pgrp,
		&session, &ttynr, &tpgid, &flags, &minflt,
		&cminflt, &majflt, &cmajflt, &utime, &stime,
	)
	if err != nil {
		return psresult{}, fmt.Errorf("Failed to parse token %v of scan output: %v", n+1, err)
	}

	// Some processes have an empty cmdline entry. In those cases, ps uses the comm field instead.
	if command == "" {
		command = comm
	}

	// statmf := ""
	// n, err = fmt.Sscanf(statm, statmf)
	activeTime := time.Duration((utime+stime)/clockTick) * time.Millisecond
	return psresult{pid: pid, active: activeTime, command: command}, nil
}

func parseStatLines(chunk string) ([]psresult, error) {
	scanner := bufio.NewScanner(strings.NewReader(chunk))
	var results []psresult
	for scanner.Scan() {
		line := scanner.Text()
		result, err := parseStatEntry(line)
		if err != nil {
			return nil, fmt.Errorf("could not parse line %v: %v", line, err)
		}

		results = append(results, result)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// PS PowerShell

func parseCsvLines(chunk string) ([]psresult, error) {
	scanner := csv.NewReader(strings.NewReader(chunk))
	scanner.Comment = '#'
	scanner.FieldsPerRecord = 3
	records, err := scanner.ReadAll()
	if err != nil {
		return nil, err
	}
	// Skip header
	records = records[1:]

	results := make([]psresult, len(records))
	for i, record := range records {
		pid, err := strconv.Atoi(record[0])
		if err != nil {
			return nil, fmt.Errorf("could not parse pid in %v: %v", record, err)
		}
		const layoutSecs = "15:04:05"
		rawTime, err := time.Parse("15:04:05.0000000", record[1])
		if err != nil {
			// Exact seconds seem to show up occasionally, so try that too.
			rawTime, err = time.Parse(layoutSecs, record[1])
			if err != nil {
				return nil, fmt.Errorf("could not parse active time as %v in %v: %v", layoutSecs, record, err)
			}
		}
		active := rawTime.Sub(time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC))
		results[i] = psresult{pid: pid, active: active, command: record[2]}
	}
	return results, nil
}

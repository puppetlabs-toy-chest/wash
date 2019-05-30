package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/cmd/internal/config"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
)

func psCommand() *cobra.Command {
	psCmd := &cobra.Command{
		Use:   "ps [<node>...]",
		Short: "Lists the processes running on the indicated compute instances",
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

// Get the list of processes separately from iterating over them. This avoids including'find' as
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

type psresult struct {
	node    string
	pid     int
	active  time.Duration
	command string
}

func parseEntry(line string) (psresult, error) {
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

func parseLines(node string, chunk string) []psresult {
	scanner := bufio.NewScanner(strings.NewReader(chunk))
	var results []psresult
	for scanner.Scan() {
		if result, err := parseEntry(scanner.Text()); err != nil {
			cmdutil.ErrPrintf("%v\n", err)
		} else {
			result.node = node
			results = append(results, result)
		}
	}
	if err := scanner.Err(); err != nil {
		cmdutil.ErrPrintf("reading standard input: %v", err)
	}
	return results
}

func formatStats(stats []psresult) string {
	headers := []cmdutil.ColumnHeader{
		{ShortName: "node", FullName: "NODE"},
		{ShortName: "pid", FullName: "PID"},
		{ShortName: "time", FullName: "TIME"},
		{ShortName: "cmd", FullName: "COMMAND"},
	}
	table := make([][]string, len(stats))
	for i, st := range stats {
		// Shorten path segments to probably-unique short strings, like `ku*s/do*p/de*t/pods/redis`.
		segments := strings.Split(strings.Trim(st.node, "/"), "/")
		for i, segment := range segments[:len(segments)-1] {
			if len(segment) > 4 {
				segments[i] = segment[:2] + "*" + segment[len(segment)-1:]
			}
		}

		table[i] = []string{
			strings.Join(segments, "/"),
			strconv.Itoa(st.pid),
			cmdutil.FormatDuration(st.active),
			st.command,
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

	conn := client.ForUNIXSocket(config.Socket)

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
			ch, err := conn.Exec(k, "sh", []string{}, apitypes.ExecOptions{Input: psScript})
			if err != nil {
				cmdutil.ErrPrintf("errored on %v: %v\n", k, err)
				return
			}
			out, err := collectOutput(ch)
			if err != nil {
				cmdutil.ErrPrintf("errored on %v: %v\n", k, err)
			} else {
				results[k] = parseLines(k, out)
			}
		}(path, i)
	}

	wg.Wait()

	var stats []psresult
	for _, path := range paths {
		stats = append(stats, results[path]...)
	}

	cmdutil.Print(formatStats(stats))
	return exitCode{0}
}

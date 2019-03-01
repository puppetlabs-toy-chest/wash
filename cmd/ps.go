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

	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/config"
)

func psCommand() *cobra.Command {
	psCmd := &cobra.Command{
		Use:   "ps [file...]",
		Short: "Lists the processes running on the indicated compute instances.",
	}

	psCmd.RunE = toRunE(psMain)

	return psCmd
}

func output(ch <-chan api.ExecPacket) (string, error) {
	exit := 0
	var stdout, stderr string
	for pkt := range ch {
		switch pktType := pkt.TypeField; pktType {
		case api.Exitcode:
			exit = int(pkt.Data.(float64))
		case api.Stdout:
			stdout += pkt.Data.(string)
		case api.Stderr:
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
		printf '%s\t%s\t%s' "$cmdline" "$(cat $i/stat)" "$(cat $i/statm)"
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

func parsePS(line string) (psresult, error) {
	tokens := strings.Split(strings.TrimSpace(line), "\t")
	if len(tokens) != 3 {
		return psresult{}, fmt.Errorf("Line did not have 3 tokens: %v", tokens)
	}
	stat := string(tokens[1])
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

	// statmf := ""
	// n, err = fmt.Sscanf(statm, statmf)
	activeTime := time.Duration((utime+stime)/clockTick) * time.Millisecond
	return psresult{pid: pid, active: activeTime, command: command}, nil
}

func parseLines(node string, chunk string) []psresult {
	scanner := bufio.NewScanner(strings.NewReader(chunk))
	var results []psresult
	for scanner.Scan() {
		if result, err := parsePS(scanner.Text()); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			result.node = node
			results = append(results, result)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
	return results
}

func formatStats(stats []psresult) string {
	headers := []columnHeader{
		{"node", "NODE"},
		{"pid", "PID"},
		{"time", "TIME"},
		{"cmd", "COMMAND"},
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
			strconv.FormatInt(int64(st.pid), 10),
			formatDuration(st.active),
			st.command,
		}
	}
	return formatTabularListing(headers, table)
}

func psMain(cmd *cobra.Command, args []string) exitCode {
	var paths []string
	if len(args) > 0 {
		paths = args
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return exitCode{1}
		}

		paths = []string{cwd}
	}

	var keys []string
	for _, path := range paths {
		apiKey, err := client.APIKeyFromPath(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to get API key for %v: %v\n", path, err)
		} else {
			keys = append(keys, apiKey)
		}
	}
	if len(keys) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no valid resources found")
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(config.Socket)

	var wg sync.WaitGroup
	wg.Add(len(keys))
	results := make(chan []psresult, len(keys))
	for i, key := range keys {
		go func(k string, idx int) {
			defer wg.Done()
			ch, err := conn.Exec(k, "sh", []string{}, api.ExecOptions{Input: psScript})
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v: %v\n", k, err)
				results <- []psresult{}
				return
			}
			out, err := output(ch)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v: %v", k, err)
				results <- []psresult{}
			} else {
				results <- parseLines(k, out)
			}
		}(key, i)
	}

	wg.Wait()

	var stats []psresult
	for range keys {
		stats = append(stats, <-results...)
	}

	fmt.Print(formatStats(stats))
	return exitCode{0}
}

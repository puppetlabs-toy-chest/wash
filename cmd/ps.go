package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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

func output(ch <-chan api.ExecPacket) (string, bool) {
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
		return fmt.Sprintf("ps exited %v\n%v", exit, stdout+stderr), false
	}
	return stdout, true
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

func parsePS(line string) string {
	tokens := strings.Split(strings.TrimSpace(line), "\t")
	if len(tokens) != 3 {
		panic(fmt.Sprintf("Should have 3 tokens from listing cmdline, stat, statm: %v", tokens))
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
		panic(fmt.Sprintf("Failed parsing token %v of scan output: %v", n+1, err))
	}

	// statmf := ""
	// n, err = fmt.Sscanf(statm, statmf)
	activeTime := time.Duration((utime+stime)/clockTick) * time.Millisecond
	return fmt.Sprintf("%v\t%v\t%v", pid, formatDuration(activeTime), command)
}

func parseLines(chunk string) string {
	scanner := bufio.NewScanner(strings.NewReader(chunk))
	output := ""
	for scanner.Scan() {
		output += parsePS(scanner.Text()) + "\n"
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
	return output
}

type result struct {
	id  string
	out string
	ok  bool
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

	// TODO: make this structured ps data.
	results := make(chan result, len(keys))
	for i, key := range keys {
		go func(k string, idx int) {
			ch, err := conn.Exec(k, "sh", []string{}, api.ExecOptions{Input: psScript})
			if err != nil {
				results <- result{k, err.Error() + "\n", false}
				return
			}
			out, ok := output(ch)
			if ok {
				results <- result{k, parseLines(out), true}
			} else {
				results <- result{k, out, false}
			}
		}(key, i)
	}

	first := true
	for range keys {
		if !first {
			fmt.Println()
		}
		first = false
		r := <-results
		fmt.Println(r.id)
		if r.ok {
			fmt.Print(r.out)
		} else {
			fmt.Fprint(os.Stderr, r.out)
		}
	}

	return exitCode{0}
}

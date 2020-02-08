package cmd

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/Benchkram/errz"
	"github.com/hpcloud/tail"
	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func tailCommand() *cobra.Command {
	tailCmd := &cobra.Command{
		Use:   "tail -f [<file>...]",
		Short: "Displays new output of files or resources with the stream action",
		Long: `Output any new updates to files and/or resources (that support the stream action). Mimics
'tail -f' for remote logs, and calls '/usr/bin/tail' if '-f' is omitted.`,
		RunE: toRunE(tailMain),
	}
	tailCmd.Flags().BoolP("follow", "f", false, "Follow new output")
	return tailCmd
}

type line struct {
	tail.Line
	source string
}

type lineWriter struct {
	name string
	buf  bytes.Buffer
	out  chan line
}

func (w *lineWriter) Write(b []byte) (int, error) {
	// Buffer lines, then submit all completed lines to the output channel. For incomplete lines
	// we just return the number of bytes written. Call Finish() when done writing to ensure any
	// final line without line endings are also written to the output channel.
	w.buf.Write(b)
	i := bytes.LastIndexAny(w.buf.Bytes(), "\r\n")
	if i == -1 {
		// Incomplete line, so just return.
		return len(b), nil
	}

	// Completed line. Remove line endings from the buffer and text (in case of \r\n) then submit it.
	text := w.buf.Next(i)

	// Consume \r or \n. Note that the Buffer takes care of re-using space when we catch up.
	crOrLf, err := w.buf.ReadByte()
	if err != nil {
		// Impossible because the next character was already found to be a \r or \n.
		panic(err)
	}

	// If the last character was \n, we could have had \r\n. We want just the line without line
	// endings so check if the previous character was \r and if so remove it.
	if last := len(text) - 1; last >= 0 && crOrLf == '\n' && text[last] == '\r' {
		text = text[:last]
	}

	w.out <- line{Line: tail.Line{Text: string(text), Time: time.Now()}, source: w.name}
	return len(b), nil
}

func (w *lineWriter) Finish() {
	if w.buf.Len() > 0 {
		// Write remainder because it didn't end in a newline.
		w.out <- line{Line: tail.Line{Text: w.buf.String(), Time: time.Now()}, source: w.name}
	}
}

// Streams output via API to aggregator channel.
// Returns nil if streaming's not supported on this path.
func tailStream(conn client.Client, agg chan line, path string) io.Closer {
	stream, err := conn.Stream(path)
	if err != nil {
		if errObj, ok := err.(*apitypes.ErrorObj); ok {
			if errObj.Kind == apitypes.UnsupportedAction {
				// The resource exists but does not support the streaming action
				return nil
			}
			cmdutil.ErrPrintf("%v\n", errObj.Msg)
		} else {
			cmdutil.ErrPrintf("%v\n", err)
		}
		return ioutil.NopCloser(nil)
	}

	// Start copying the stream to the aggregate channel
	go func() {
		lw := lineWriter{name: path, out: agg}
		_, err := io.Copy(&lw, stream)
		if err != nil {
			agg <- line{Line: tail.Line{Time: time.Now(), Err: err}, source: path}
		} else {
			lw.Finish()
		}
	}()
	return stream
}

var endOfFileLocation = tail.SeekInfo{Offset: 0, Whence: 2}

type tailCloser struct{ *tail.Tail }

func (c tailCloser) Close() error {
	c.Cleanup()
	return c.Stop()
}

func tailFile(agg chan line, path string) io.Closer {
	// Error handling here mimics linux 'tail': it prints an error and continues for any other
	// input. Note that the 'tail' package we use doesn't emit anything when it's called on a
	// directory or non-existant file.
	if finfo, err := os.Stat(path); err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return ioutil.NopCloser(nil)
	} else if finfo.IsDir() {
		cmdutil.ErrPrintf("tail %v: is a directory\n", path)
		return ioutil.NopCloser(nil)
	}

	// Set Location so we start streaming at the end of the file
	tailer, err := tail.TailFile(path, tail.Config{
		Follow:   true,
		Location: &endOfFileLocation,
		Logger:   tail.DiscardingLogger,
	})
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return ioutil.NopCloser(nil)
	}

	// Start copying the tail to the aggregate channel
	go func() {
		for ln := range tailer.Lines {
			agg <- line{Line: *ln, source: path}
		}
	}()
	return tailCloser{tailer}
}

func tailMain(cmd *cobra.Command, args []string) exitCode {
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		panic(err.Error())
	}

	if !follow {
		// Defer to `/usr/bin/tail`
		comm := exec.Command("/usr/bin/tail", args...)
		comm.Stdin = os.Stdin
		comm.Stdout = os.Stdout
		comm.Stderr = os.Stderr
		if err := comm.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitCode{exitErr.ExitCode()}
			}
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}
		return exitCode{0}
	}

	// If no paths are declared, try to stream the current directory/resource
	if len(args) == 0 {
		args = []string{"."}
	}

	conn := cmdutil.NewClient()
	agg := make(chan line)

	// Try streaming as a resource, then as a file if that failed for predictable reasons
	for _, path := range args {
		if closer := tailStream(conn, agg, path); closer != nil {
			defer func() { errz.Log(closer.Close()) }()
			continue
		}

		// Unable to read as a stream, try as a file.
		if closer := tailFile(agg, path); closer != nil {
			defer func() { errz.Log(closer.Close()) }()
		}
	}

	// Print from aggregate channel
	var last string
	for ln := range agg {
		if ln.Err != nil {
			cmdutil.ErrPrintf("%v\n", ln.source, ln.Err)
			continue
		}

		if last != ln.source {
			if last != "" {
				// Leave a space before changing sources
				cmdutil.Println()
			}
			last = ln.source
			cmdutil.Println("===>", last, "<===")
		}

		cmdutil.Println(ln.Text)
	}

	return exitCode{0}
}

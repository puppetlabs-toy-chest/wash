package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/Benchkram/errz"
	"github.com/hpcloud/tail"
	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
)

func tailCommand() *cobra.Command {
	tailCmd := &cobra.Command{
		Use:   "tail -f [<file>...]",
		Short: "Displays new output of files or resources that support the stream action",
		Long: `Output any new updates to files and/or resources (that support the stream action). Currently
requires the '-f' option to run. Attempts to mimic the functionality of 'tail -f' for remote logs.`,
		RunE: toRunE(tailMain),
	}
	tailCmd.Flags().BoolP("follow", "f", false, "Follow new output (required)")
	return tailCmd
}

type line struct {
	tail.Line
	source string
}

type lineWriter struct {
	name string
	out  chan line
}

func (w lineWriter) Write(b []byte) (int, error) {
	s := strings.TrimSuffix(string(b), "\n")
	w.out <- line{Line: tail.Line{Text: s, Time: time.Now()}, source: w.name}
	return len(b), nil
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
		_, err := io.Copy(lineWriter{name: path, out: agg}, stream)
		if err != nil {
			agg <- line{Line: tail.Line{Time: time.Now(), Err: err}, source: path}
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
		cmdutil.ErrPrintf("Please use -f, other operations are not yet supported\n")
		return exitCode{1}
	}

	// If no paths are declared, try to stream the current directory/resource
	if len(args) == 0 {
		args = []string{"."}
	}

	conn := client.ForUNIXSocket(config.Socket)
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
				fmt.Println()
			}
			last = ln.source
			fmt.Println("===>", last, "<===")
		}

		fmt.Println(ln.Text)
	}

	return exitCode{0}
}

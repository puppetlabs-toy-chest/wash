package cmd

import (
	"fmt"
	"io"
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
	"github.com/spf13/viper"
)

func tailCommand() *cobra.Command {
	tailCmd := &cobra.Command{
		Use:   "tail -f <file> [<file>...]",
		Short: "Displays new output of files or resources that support the stream action",
	}

	tailCmd.Flags().BoolP("follow", "f", false, "Follow new output (required)")
	if err := viper.BindPFlag("follow", tailCmd.Flags().Lookup("follow")); err != nil {
		cmdutil.ErrPrintf("%v\n", err)
	}

	tailCmd.RunE = toRunE(tailMain)

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

func tailMain(cmd *cobra.Command, args []string) exitCode {
	follow := viper.GetBool("follow")
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

	// Separate paths into streamable resources and files
	var files []string
	for _, path := range args {
		apiPath, err := client.APIKeyFromPath(path)
		if err != nil {
			// Not a resource
			files = append(files, path)
			continue
		}

		if stream, err := conn.Stream(apiPath); err == nil {
			defer func() { errz.Log(stream.Close()) }()
			// Start copying the stream to the aggregate channel
			go func(src string) {
				_, err := io.Copy(lineWriter{name: src, out: agg}, stream)
				if err != nil {
					agg <- line{Line: tail.Line{Time: time.Now(), Err: err}, source: src}
				}
			}(path)
		} else {
			if errObj, ok := err.(*apitypes.ErrorObj); ok {
				if errObj.Kind == "puppetlabs.wash/unsupported-action" {
					// The resource exists but does not support the streaming action, try to read it as a file.
					files = append(files, path)
				} else {
					cmdutil.ErrPrintf("%v\n", errObj.Msg)
				}
			} else {
				cmdutil.ErrPrintf("%v\n", err)
			}
		}
	}

	// Set Location so we start streaming at the end of the file
	tailConf := tail.Config{
		Follow:   true,
		Location: &tail.SeekInfo{Offset: 0, Whence: 2},
		Logger:   tail.DiscardingLogger,
	}
	for _, path := range files {
		// Error handling here mimics linux 'tail': it prints an error and continues for any other
		// input. Note that the 'tail' package we use doesn't emit anything when it's called on a
		// directory or non-existant file.
		if finfo, err := os.Stat(path); err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			continue
		} else if finfo.IsDir() {
			cmdutil.ErrPrintf("tail %v: is a directory\n", path)
			continue
		}

		tailer, err := tail.TailFile(path, tailConf)
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			continue
		}

		defer func() {
			errz.Log(tailer.Stop())
			tailer.Cleanup()
		}()
		// Start copying the tail to the aggregate channel
		go func(src string) {
			for ln := range tailer.Lines {
				agg <- line{Line: *ln, source: src}
			}
		}(path)
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

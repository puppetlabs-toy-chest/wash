package cmd

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jedib0t/go-pretty/progress"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func validateCommand() *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate <plugin>",
		Short: "Validates an external plugin",
		Long: `Validates an external plugin, using it's schema to limit exploration. The plugin can be one you've
configured in Wash's config file, or it can be a script to load as an external plugin. Plugin-
specific config from Wash's config file will be used. The Wash daemon does not need to be running
to use this command.

Validate starts from the plugin root and does a breadth-first traversal of the plugin hierarchy,
invoking all supported methods on an example at each level. If the plugin provides a schema, it
will be used to explore one example of each type of entry. Exploration can be stopped with Ctrl-C
when needed.

Each line represents validation of an entry type. The 'lrsx' fields represent support for 'list',
'read', 'stream', and 'execute' methods respectively, with '-' representing lack of support for a
method.`,
		Args:   cobra.ExactArgs(1),
		PreRun: bindServerArgs,
		RunE:   toRunE(validateMain),
	}
	validateCmd.Flags().IntP("parallel", "p", 10, "Number of entries to validate in parallel")
	validateCmd.Flags().BoolP("all", "a", false, "Validate all entries rather than an example at each level of hierarchy")
	addServerArgs(validateCmd, "warn")
	return validateCmd
}

func validateMain(cmd *cobra.Command, args []string) exitCode {
	// Validate that 'plugin' is a valid plugin and load it
	plugins, serverOpts, err := serverOptsFor(cmd)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	parallel, err := cmd.Flags().GetInt("parallel")
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	plug := args[0]
	root, ok := plugins[plug]
	if !ok {
		// See if it's a script we can run as an external plugin instead
		spec := plugin.ExternalPluginSpec{Script: plug}
		root, err = spec.Load()
		if err != nil {
			pluginNames := make([]string, 0, len(plugins))
			for name := range plugins {
				pluginNames = append(pluginNames, name)
			}
			msg := "Unable to load %v as a configured plugin or script: %v\nValid plugins are: %v\n"
			cmdutil.ErrPrintf(msg, plug, err, strings.Join(pluginNames, ", "))
			return exitCode{1}
		}
	}

	// Configure logging
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	logFH, err := serverOpts.SetupLogging()
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}
	if logFH != nil {
		defer logFH.Close()
	}

	registry := plugin.NewRegistry()
	if err := registry.RegisterPlugin(root, serverOpts.PluginConfig[plug]); err != nil {
		cmdutil.ErrPrintf("%v\n", formatErr("Error loading plugin", "init", err))
		return exitCode{1}
	}

	rand.Seed(time.Now().UnixNano())
	plugin.InitCache()
	var wg sync.WaitGroup
	wg.Add(2)

	pw := progress.NewWriter()
	pw.SetUpdateFrequency(50 * time.Millisecond)
	pw.Style().Colors = progress.StyleColorsExample

	go func() {
		pw.Render()
		wg.Done()
	}()

	// On Ctrl-C cancel the context to ensure plugin calls have a chance to cleanup.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sigCh
		cancel()
	}()

	// Use list to walk the hierarchy breadth-first. If a schema is present, visit each unique type
	// at each level once, and stop when we've fully explored the schema. If there are errors, print
	// all the detail we have about them. Use an unbuffered channel to register errors as they come in.
	// Breadth-first walk is handled by recursive calls to processEntry, which can be run in parallel
	// with a worker pool.
	erred := 0
	errs := make(chan error)
	go func() {
		for err := range errs {
			erred++
			cmdutil.ErrPrintf("%v\n", err)
		}
		wg.Done()
	}()

	// Use CachedList on the registry to ensure cache IDs are generated.
	entries, err := plugin.CachedList(ctx, registry)
	if err != nil {
		panic("CachedList on registry should not fail")
	}

	// We use a worker pool to limit work-in-progress. Put the plugin on the worker pool.
	wp := cmdutil.NewPool(parallel)
	for _, e := range entries {
		wp.Submit(func() { processEntry(ctx, pw, wp, e, all, errs) })
	}

	// Wait for work to complete.
	wp.Finish()

	// Leave time for progress to finish rendering.
	time.Sleep(100 * time.Millisecond)
	pw.Stop()

	// All error generators should be done. Close the channel and wait for the error processing
	// routine to complete.
	close(errs)
	wg.Wait()
	if erred > 0 {
		cmdutil.ErrPrintf("Found %v errors.\n", erred)
		return exitCode{1}
	}
	cmdutil.Println("Looks good!")
	return exitCode{0}
}

// If the entry has a schema, use it to help further distinguish between different things that
// behave the same.
type criteria struct {
	label, typeID            string
	list, read, stream, exec bool
	singleton                bool
}

func newCriteria(entry plugin.Entry) criteria {
	var crit criteria
	if schema := entry.Schema(); schema != nil {
		crit.label = schema.Label
		crit.typeID = schema.TypeID
		crit.singleton = schema.Singleton
	}
	crit.list = plugin.ListAction().IsSupportedOn(entry)
	crit.read = plugin.ReadAction().IsSupportedOn(entry)
	crit.stream = plugin.StreamAction().IsSupportedOn(entry)
	crit.exec = plugin.ExecAction().IsSupportedOn(entry)
	return crit
}

func (c criteria) String() string {
	s := []byte("----")
	if c.list {
		s[0] = 'l'
	}

	if c.read {
		s[1] = 'r'
	}

	if c.stream {
		s[2] = 's'
	}

	if c.exec {
		s[3] = 'x'
	}

	result := string(s)
	if c.label != "" {
		label := c.label
		if !c.singleton {
			label = "[" + label + "]"
		}
		result = fmt.Sprintf("%s %-20s", result, label)
	}
	return result
}

var timeoutDuration = 30 * time.Second

// Invokes 'fn' with a context with timeout. Invokes the cancelFunc on error.
func withTimeout(ctx context.Context, method, name string,
	fn func(context.Context) (interface{}, error)) (interface{}, context.CancelFunc, error) {
	limitedCtx, cancelFunc := context.WithTimeout(ctx, timeoutDuration)
	obj, err := fn(limitedCtx)

	var cancelled bool
	select {
	case <-limitedCtx.Done():
		cancelled = true
	default:
	}

	if err != nil {
		cancelFunc()
		msg := fmt.Sprintf("Error validating %v on %v", strings.Title(method), name)
		if cancelled {
			select {
			case <-ctx.Done():
				msg += ", shutting down after Ctrl-C"
			default:
				msg = fmt.Sprintf("%v, operation timed out after %v", msg, timeoutDuration)
			}
		}
		return nil, nil, formatErr(msg, method, err)
	}
	return obj, cancelFunc, nil
}

func processEntry(ctx context.Context, pw progress.Writer, wp cmdutil.Pool, e plugin.Entry, all bool, errs chan<- error) {
	defer wp.Done()
	name := plugin.ID(e)
	crit := newCriteria(e)
	tracker := progress.Tracker{Message: fmt.Sprintf("Testing %s %s", crit, name), Total: 4}
	pw.AppendTracker(&tracker)

	if plugin.ListAction().IsSupportedOn(e) {
		obj, cancelFunc, err := withTimeout(ctx, "list", name, func(ctx context.Context) (interface{}, error) {
			return plugin.CachedList(ctx, e.(plugin.Parent))
		})
		if err != nil {
			errs <- err
			return
		}
		cancelFunc()
		entries := obj.(map[string]plugin.Entry)

		if all {
			for _, entry := range entries {
				// Make a local copy for the lambda to capture.
				entry := entry
				wp.Submit(func() { processEntry(ctx, pw, wp, entry, all, errs) })
			}
		} else {
			// Group children by ones that look "similar", and select one from each group to test.
			groups := make(map[criteria][]plugin.Entry)
			for _, entry := range entries {
				ccrit := newCriteria(entry)
				// If we have a TypeID, only explore children if they are different from the parent.
				// This prevents simple recursion like volume directories containing more dirs.
				if ccrit.typeID == "" || ccrit != crit {
					groups[ccrit] = append(groups[ccrit], entry)
				}
			}

			for _, items := range groups {
				entry := items[rand.Intn(len(items))]
				wp.Submit(func() { processEntry(ctx, pw, wp, entry, all, errs) })
			}
		}
	}
	tracker.Increment(1)

	if plugin.ReadAction().IsSupportedOn(e) {
		_, cancelFunc, err := withTimeout(ctx, "read", name, func(ctx context.Context) (interface{}, error) {
			return plugin.CachedOpen(ctx, e.(plugin.Readable))
		})
		if err != nil {
			errs <- err
			return
		}
		cancelFunc()
	}
	tracker.Increment(1)

	if plugin.StreamAction().IsSupportedOn(e) {
		obj, cancelFunc, err := withTimeout(ctx, "stream", name, func(ctx context.Context) (interface{}, error) {
			return e.(plugin.Streamable).Stream(ctx)
		})
		if err != nil {
			errs <- err
			return
		}
		obj.(io.Closer).Close()
		cancelFunc()
	}
	tracker.Increment(1)

	if plugin.ExecAction().IsSupportedOn(e) {
		const testMessage = "hello"
		obj, cancelFunc, err := withTimeout(ctx, "exec", name, func(ctx context.Context) (interface{}, error) {
			return e.(plugin.Execable).Exec(ctx, "echo", []string{testMessage}, plugin.ExecOptions{})
		})
		if err != nil {
			errs <- err
			return
		}
		cmd := obj.(plugin.ExecCommand)

		var output string
		for chunk := range cmd.OutputCh() {
			if err := chunk.Err; err != nil {
				errs <- err
			} else if chunk.StreamID == plugin.Stdout {
				output += chunk.Data
			} else if chunk.StreamID == plugin.Stderr {
				errs <- fmt.Errorf("Unexpected error output on Exec: %v", chunk.Data)
			}
		}

		if msg := strings.Trim(output, "\n"); msg != testMessage {
			errs <- fmt.Errorf("Unexpected output on Exec: %v", msg)
		}

		if exitCode, err := cmd.ExitCode(); err != nil {
			errs <- fmt.Errorf("Error getting exit code for 'echo': %v", err)
		} else if exitCode != 0 {
			errs <- fmt.Errorf("Non-zero exit code for 'echo': %v", exitCode)
		}
		cancelFunc()
	}
	tracker.MarkAsDone()
}

func formatErr(msg, method string, err error) error {
	helpURL := "https://puppetlabs.github.io/wash/docs/external_plugins/#" + method
	return fmt.Errorf("%v: %v\nSee %v for response format", msg, err, helpURL)
}

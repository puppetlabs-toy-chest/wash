package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/gammazero/workerpool"
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
	ctx := context.Background()

	pw := progress.NewWriter()
	pw.SetUpdateFrequency(50 * time.Millisecond)
	pw.Style().Colors = progress.StyleColorsExample
	go pw.Render()

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
	}()

	// Use CachedList on the registry to ensure cache IDs are generated.
	entries, err := plugin.CachedList(ctx, registry)
	if err != nil {
		panic("CachedList on registry should not fail")
	}

	// We use a worker pool to limit work-in-progress. Put the plugin on the worker pool.
	wp := newPool(parallel)
	for _, e := range entries {
		wp.Submit(func() { processEntry(ctx, pw, wp, e, all, errs) })
	}

	// Wait for work to complete.
	wp.Finish()

	// Leave time for progress to finish rendering.
	time.Sleep(100 * time.Millisecond)
	pw.Stop()

	if erred > 0 {
		cmdutil.ErrPrintf("Found %v errors.\n", erred)
		return exitCode{1}
	}
	cmdutil.Println("Looks good!")
	return exitCode{0}
}

// We use a worker pool to limit work-in-progress, and a wait group to know when all queued work
// is complete. Because the queue is dynamic, we need a wait group to tell us when all potential
// work is complete before we tell the pool top stop accepting new work and shutdown.
type pool struct {
	wp *workerpool.WorkerPool
	wg *sync.WaitGroup
}

func newPool(parallel int) pool {
	return pool{wp: workerpool.New(parallel), wg: &sync.WaitGroup{}}
}

func (p pool) Submit(f func()) {
	p.wg.Add(1)
	p.wp.Submit(f)
}

func (p pool) Done() {
	p.wg.Done()
}

func (p pool) Finish() {
	// Wait for the workgroup's we've queued to finish, then stop the worker pool.
	p.wg.Wait()
	p.wp.StopWait()
}

type criteria struct {
	list, read, stream, exec bool
}

func newCriteria(entry plugin.Entry) criteria {
	var crit criteria
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
	return string(s)
}

func processEntry(ctx context.Context, pw progress.Writer, wp pool, e plugin.Entry, all bool, errs chan<- error) {
	defer wp.Done()
	name := plugin.ID(e)
	crit := newCriteria(e)
	tracker := progress.Tracker{Message: fmt.Sprintf("Testing %s %s", crit, name), Total: 4}
	pw.AppendTracker(&tracker)

	if plugin.ListAction().IsSupportedOn(e) {
		entries, err := plugin.CachedList(ctx, e.(plugin.Parent))
		if err != nil {
			errs <- formatErr(fmt.Sprintf("Error validating List on %v", name), "list", err)
			return
		}

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
				crit := newCriteria(entry)
				groups[crit] = append(groups[crit], entry)
			}

			for _, items := range groups {
				entry := items[rand.Intn(len(items))]
				wp.Submit(func() { processEntry(ctx, pw, wp, entry, all, errs) })
			}
		}
	}
	tracker.Increment(1)

	if plugin.ReadAction().IsSupportedOn(e) {
		_, err := plugin.CachedOpen(ctx, e.(plugin.Readable))
		if err != nil {
			errs <- formatErr(fmt.Sprintf("Error validating Read on %v", name), "read", err)
			return
		}
	}
	tracker.Increment(1)

	if plugin.StreamAction().IsSupportedOn(e) {
		rdr, err := e.(plugin.Streamable).Stream(ctx)
		if err != nil {
			errs <- formatErr(fmt.Sprintf("Error validating Stream on %v", name), "stream", err)
			return
		}
		rdr.Close()
	}
	tracker.Increment(1)

	// TODO: decide how to test exec. 'echo' is pretty portable.
	//if plugin.ExecAction().IsSupportedOn(e) {}
	tracker.MarkAsDone()
}

func formatErr(msg, method string, err error) error {
	helpURL := "https://puppetlabs.github.io/wash/docs/external_plugins/#" + method
	return fmt.Errorf("%v: %v\nSee %v for response format", msg, err, helpURL)
}

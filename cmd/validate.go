package cmd

import (
	"context"
	"strings"

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
invoking all supported methods on examples at each level. If the plugin provides a schema, it will
be used to limit exploration to one example of each type of entry. Exploration can be stopped with
Ctrl-C when needed.`,
		Args:   cobra.ExactArgs(1),
		PreRun: bindServerArgs,
		RunE:   toRunE(validateMain),
	}
	addServerArgs(validateCmd, "info")
	return validateCmd
}

func validateMain(cmd *cobra.Command, args []string) exitCode {
	// Validate that 'plugin' is a valid plugin and load it
	plugins, serverOpts, err := serverOptsFor(cmd)
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
		cmdutil.ErrPrintf("Error loading plugin: %v\n", err)
		return exitCode{1}
	}

	plugin.InitCache()
	ctx := context.Background()

	// Use list to walk the hierarchy breadth-first. If a schema is present, visit each unique type
	// at each level once, and stop when we've fully explored the schema. If there are errors, print
	// all the detail we have about them.
	queue := make([]plugin.Parent, 0)
	queue = append(queue, registry)
	for len(queue) > 0 {
		top := queue[0]
		queue = queue[1:]
		entries, err := plugin.CachedList(ctx, top)
		if err != nil {
			cmdutil.ErrPrintf("Error Listing %v: %v\n", plugin.Name(top), err)
			return exitCode{1}
		}

		for _, entry := range entries {
			if plugin.ListAction().IsSupportedOn(entry) {
				queue = append(queue, entry.(plugin.Parent))
			}
		}
	}

	cmdutil.Println("Looks good!")
	return exitCode{0}
}

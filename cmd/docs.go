package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kballard/go-shellquote"
	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
	"github.com/spf13/cobra"
)

func docsCommand() *cobra.Command {
	docsCmd := &cobra.Command{
		Use:   "docs <path>",
		Short: "Displays the entry's documentation",
		RunE:  toRunE(docsMain),
	}
	return docsCmd
}

func docsMain(cmd *cobra.Command, args []string) exitCode {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	conn := cmdutil.NewClient()

	entry, err := conn.Info(path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}
	schema, err := conn.Schema(path)
	if err != nil {
		cmdutil.ErrPrintf("failed to get the schema: %v\n", err)
		return exitCode{1}
	}

	docs := &strings.Builder{}

	// Print the description
	if schema != nil && len(schema.Description()) > 0 {
		addSection(docs, strings.Trim(schema.Description(), "\n"))
	}

	if mountpoint := os.Getenv("W"); len(mountpoint) > 0 {
		if strings.HasPrefix(path, mountpoint) {
			// Munge the path so that something like 'docs $W/gcp'
			// will generate examples like 'ls $W/gcp' instead of
			// 'ls <mountpoint>/gcp'.
			path = "$W" + strings.TrimPrefix(path, mountpoint)
		}
	}

	// Print the supported attributes. This part is printed as
	//   SUPPORTED ATTRIBUTES
	//     * <attribute> (<full_name_of_attribute>)
	//
	//   <description that talks about attributes/metadata and shows off 'meta'>
	//
	// if the entry has any supported attributes. Otherwise, it prints an appropriate
	// note then prints the attributes/metadata description.
	//
	// NOTE: If the entry has attributes but doesn't have any partial metadata, then
	// entry.Metadata contains all of the attributes so this check still works. See
	// plugin.PartialMetadata's comments for more details.
	if len(entry.Metadata) > 0 {
		addSection(docs, stringifySupportedAttributes(path, entry))
	}

	// Print the supported actions. This part is printed as
	//   SUPPORTED ACTIONS
	//     * <action>
	//         <description>
	if len(entry.Actions) > 0 {
		addSection(docs, stringifySupportedActions(path, entry))
	}

	// Print the supported signals/signal groups (if there are any). This part is
	// printed as
	//   SUPPORTED SIGNALS
	//     * <signal>
	//         <desc>
	//     * <signal>
	//         <desc>
	//
	//   SUPPORTED SIGNAL GROUPS
	//     * <signal_group>
	//         <desc>
	//     * <signal_group>
	//         <desc>
	if schema != nil && len(schema.Signals()) > 0 {
		var supportedSignals []apitypes.SignalSchema
		var supportedSignalGroups []apitypes.SignalSchema
		for _, signalSchema := range schema.Signals() {
			if signalSchema.IsGroup() {
				supportedSignalGroups = append(supportedSignalGroups, signalSchema)
			} else {
				supportedSignals = append(supportedSignals, signalSchema)
			}
		}
		if len(supportedSignals) > 0 {
			addSection(docs, stringifySignalSet("SUPPORTED SIGNALS", supportedSignals))
		}
		if len(supportedSignalGroups) > 0 {
			addSection(docs, stringifySignalSet("SUPPORTED SIGNAL GROUPS", supportedSignalGroups))
		}
	}

	cmdutil.Println(docs.String())
	return exitCode{0}
}

func stringifySupportedAttributes(path string, entry apitypes.Entry) string {
	path = shellquote.Join(path)
	var supportedAttributes strings.Builder
	supportedAttributes.WriteString("SUPPORTED ATTRIBUTES\n")
	if len(entry.Attributes.ToMap()) <= 0 {
		lines := []string{
			fmt.Sprintf("This entry hasn't specified any attributes. However, it does have some metadata."),
		}
		supportedAttributes.WriteString(strings.Join(lines, "\n"))
	} else {
		for attr, value := range entry.Attributes.ToMap() {
			supportedAttributes.WriteString(fmt.Sprintf("* %v", attr))
			var fullAttrName string
			switch attr {
			case "atime":
				fullAttrName = "last access time"
			case "mtime":
				fullAttrName = "last modified time"
			case "ctime":
				fullAttrName = "change time"
			case "crtime":
				fullAttrName = "creation time"
			}
			if len(fullAttrName) > 0 {
				supportedAttributes.WriteString(fmt.Sprintf(" (%v)", fullAttrName))
			}
			supportedAttributes.WriteString(fmt.Sprintf(" -- %s\n", value))
		}
	}
	supportedAttributes.WriteString("\n")
	metadataLines := []string{
		fmt.Sprintf("The attributes are a subset of the entry's metadata, which contains everything"),
		fmt.Sprintf("you ever need to know about the entry. You can use"),
		fmt.Sprintf("    meta %s", path),
		fmt.Sprintf("to view the metadata and"),
		fmt.Sprintf("    meta --partial %s", path),
		fmt.Sprintf("to view the partial metadata. You can use 'find' to filter entries on their"),
		fmt.Sprintf("attributes and metadata. Type 'find --help' to see all the properties that"),
		fmt.Sprintf("you can filter on."),
	}
	supportedAttributes.WriteString(strings.Join(metadataLines, "\n"))
	return supportedAttributes.String()
}

func stringifySupportedActions(path string, entry apitypes.Entry) string {
	path = shellquote.Join(path)
	var supportedActions strings.Builder
	supportedActions.WriteString("SUPPORTED ACTIONS\n")
	actions := entry.Actions
	sort.Strings(actions)
	for _, action := range actions {
		supportedActions.WriteString(fmt.Sprintf("* %v\n", action))
		var actionDescriptionLines []string
		switch action {
		case plugin.ListAction().Name:
			actionDescriptionLines = []string{
				fmt.Sprintf("- ls %s", path),
				fmt.Sprintf("    Type 'docs <child>' to view an ls'ed child's documentation"),
				fmt.Sprintf("- cd %s", path),
				fmt.Sprintf("    The 'W' environment variable contains the Wash root. Use 'cd $W' to"),
				fmt.Sprintf("    return to it."),
				fmt.Sprintf("- stree %s", path),
				fmt.Sprintf("    Gives you a high-level overview of the kinds of things that you will"),
				fmt.Sprintf("    encounter when you 'cd' and 'ls' through this entry."),
				fmt.Sprintf("- (anything else that works with directories [e.g. 'tree'])"),
			}
		case plugin.ReadAction().Name:
			actionDescriptionLines = []string{
				fmt.Sprintf("- cat %s", path),
				fmt.Sprintf("- grep 'foo' %s", path),
				fmt.Sprintf("- (anything else that reads files [e.g. 'less'])"),
			}
		case plugin.StreamAction().Name:
			actionDescriptionLines = []string{
				fmt.Sprintf("- tail -f %s", path),
			}
		case plugin.WriteAction().Name:
			if entry.Attributes.HasSize() && entry.Supports(plugin.ReadAction()) {
				// Entry is file-like so include file-like examples
				actionDescriptionLines = []string{
					fmt.Sprintf("- echo 'foo' > %s", path),
					fmt.Sprintf("    Overwrites the file with 'foo'"),
					fmt.Sprintf("- echo 'foo' >> %s", path),
					fmt.Sprintf("    Appends 'foo' to the file"),
					fmt.Sprintf("- vim %s", path),
					fmt.Sprintf("    Edits the file with 'vim'"),
					fmt.Sprintf("- (anything else that lets you write/edit files)"),
				}
			} else {
				actionDescriptionLines = []string{
					fmt.Sprintf("- echo 'foo' > %s", path),
					fmt.Sprintf("- echo 'foo' >> %s", path),
					fmt.Sprintf("    Both commands write the chunk 'foo' to the entry"),
					fmt.Sprintf("- (anything else that writes files [excluding editors like 'vim'])"),
				}
			}
		case plugin.ExecAction().Name:
			actionDescriptionLines = []string{
				fmt.Sprintf("- wexec %s <command> <args...>", path),
				fmt.Sprintf("    e.g. wexec %s uname", path),
			}
		case plugin.DeleteAction().Name:
			actionDescriptionLines = []string{
				fmt.Sprintf("- delete %s", path),
			}
		case plugin.SignalAction().Name:
			actionDescriptionLines = []string{
				fmt.Sprintf("- signal <signal> %s", path),
				fmt.Sprintf("    e.g. signal start %s", path),
			}
		}
		for _, line := range actionDescriptionLines {
			supportedActions.WriteString(fmt.Sprintf("    %v\n", line))
		}
	}
	supportedActions.WriteString("\nType 'help' to see a list of all the built-in Wash commands.")
	return supportedActions.String()
}

func stringifySignalSet(setName string, signals []apitypes.SignalSchema) string {
	var signalSet strings.Builder
	signalSet.WriteString(fmt.Sprintf("%v\n", setName))
	for _, signal := range signals {
		signalSet.WriteString(fmt.Sprintf("* %v\n", signal.Name()))
		lines := strings.Split(strings.Trim(signal.Description(), "\n"), "\n")
		for _, line := range lines {
			signalSet.WriteString(fmt.Sprintf("    %v\n", line))
		}
	}
	return signalSet.String()
}

func addSection(docs *strings.Builder, section string) {
	section = strings.Trim(section, "\n")
	if docs.Len() > 0 {
		_, _ = docs.WriteString("\n\n")
	}
	docs.WriteString(section)
}

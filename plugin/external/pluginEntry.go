package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/getlantern/deepcopy"
	"github.com/jinzhu/copier"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/transport"
)

type decodedCacheTTLs struct {
	List     time.Duration `json:"list"`
	Read     time.Duration `json:"read"`
	Metadata time.Duration `json:"metadata"`
}

// decodedExternalPluginEntry describes a decoded serialized entry.
// Methods can be a string, or a tuple of [<method>, <value>]. <value> can take different forms
// depending on the method. json.RawMessage lets us delay parsing these so we can parse them into
// method-specific structs, which greatly simplifies validation.
type decodedExternalPluginEntry struct {
	TypeID             string                 `json:"type_id"`
	Name               string                 `json:"name"`
	Methods            []json.RawMessage      `json:"methods"`
	SlashReplacer      string                 `json:"slash_replacer"`
	CacheTTLs          decodedCacheTTLs       `json:"cache_ttls"`
	InaccessibleReason string                 `json:"inaccessible_reason"`
	Attributes         plugin.EntryAttributes `json:"attributes"`
	PartialMetadata    plugin.JSONObject      `json:"partial_metadata"`
	State              string                 `json:"state"`
}

type methodTuple struct {
	Method string
	Value  json.RawMessage
}

func (m *methodTuple) UnmarshalJSON(data []byte) error {
	var list []json.RawMessage
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	if len(list) != 2 {
		return fmt.Errorf("each method must be a string or tuple [<method>, <value>], not %v", string(data))
	}

	if err := json.Unmarshal(list[0], &m.Method); err != nil {
		return err
	}
	m.Value = list[1]
	return nil
}

type execImpl struct {
	Transport string             `json:"transport"`
	Options   transport.Identity `json:"options"`
}

func (e decodedExternalPluginEntry) getMungedMethods() (map[string]methodInfo, error) {
	methods := make(map[string]methodInfo)
	for _, raw := range e.Methods {
		// Try to unmarshal to a string. If that doesn't work, unmarshal to a method-specific tuple.
		var name string
		if err := json.Unmarshal(raw, &name); err == nil {
			methods[name] = methodInfo{signature: plugin.DefaultSignature}
			continue
		}

		var tuple methodTuple
		if err := json.Unmarshal(raw, &tuple); err != nil {
			return nil, err
		}

		// Assume a default signature. Then validate prefetched values for specific methods and update
		// methodInfo as needed.
		info := methodInfo{signature: plugin.DefaultSignature}
		switch tuple.Method {
		case "read":
			// Check if we have ["read", <block_readable?>] or ["read", <content>].
			// The latter implies <block_readable> == false.
			var blockReadable bool
			if err := json.Unmarshal(tuple.Value, &blockReadable); err == nil {
				if blockReadable {
					info.signature = plugin.BlockReadableSignature
				}
				break
			}

			var content string
			if err := json.Unmarshal(tuple.Value, &content); err != nil {
				return nil, fmt.Errorf("Read method must provide a string, not %v", string(tuple.Value))
			}
			info.tupleValue = []byte(content)
		case "list":
			var decodedEntries []decodedExternalPluginEntry
			if err := json.Unmarshal(tuple.Value, &decodedEntries); err != nil {
				return nil, fmt.Errorf("implementation of list must conform to %v, not %v", listFormat, string(tuple.Value))
			}
			info.tupleValue = decodedEntries
		case "schema":
			graph, err := unmarshalSchemaGraph(e.Name, e.TypeID, tuple.Value)
			if err != nil {
				return nil, fmt.Errorf(
					"could not decode schema from stdout: %v\nreceived:\n%v\nexpected something like:\n%v",
					err,
					string(tuple.Value),
					schemaFormat,
				)
			}
			info.tupleValue = graph
		case "exec":
			// Check if we have ["exec", <exec_implementation>].
			var impl execImpl
			if err := json.Unmarshal(tuple.Value, &impl); err != nil {
				return nil, fmt.Errorf("result for exec must specify an implementation transport and options")
			} else if impl.Transport != "ssh" {
				return nil, fmt.Errorf("unsupported transport %v requested, only ssh is supported", impl.Transport)
			}
			info.tupleValue = impl
		}

		methods[tuple.Method] = info
	}
	return methods, nil
}

func (e decodedExternalPluginEntry) toExternalPluginEntry(ctx context.Context, schemaKnown, isRoot bool) (*pluginEntry, error) {
	if len(e.Name) <= 0 {
		return nil, fmt.Errorf("the entry name must be provided")
	}
	if e.Methods == nil {
		return nil, fmt.Errorf("the entry's methods must be provided")
	}

	methods, err := e.getMungedMethods()
	if err != nil {
		return nil, err
	}

	// INVARIANT: If root implements schema, then schemaKnown == true (and vice versa).
	// Idea here is that entry schemas also include their descendant's schema. So if the
	// root implements schema, then the root's schema will include every entry's schema.
	// Thus, it is reasonable for us to expect (and require) every entry to implement schema
	// if the root implements it. It is also reasonable for us to expect (and require)
	// every entry to _not_ implement schema if the root does not implement it.
	if info, ok := methods["schema"]; ok {
		if len(e.TypeID) == 0 {
			return nil, fmt.Errorf("entry %v implements schema, but no type ID was provided", e.Name)
		}
		if !schemaKnown && !isRoot {
			return nil, fmt.Errorf("entry %v (%v) implements schema, but the plugin root doesn't", e.Name, e.TypeID)
		}
		// schemaKnown || isRoot
		if !isRoot && info.tupleValue != nil {
			return nil, fmt.Errorf(
				"entry %v (%v) prefetched its schema. Only plugin roots support schema prefetching",
				e.Name,
				e.TypeID,
			)
		}
		// schemaKnown || (isRoot || result == nil). In either case, the schema's
		// known.
		schemaKnown = true
	} else if schemaKnown {
		// ok == false here
		return nil, fmt.Errorf("entry %v (%v) must implement schema", e.Name, e.TypeID)
	}

	// If read content is static, it's likely it's not coming from a source that separately provides
	// the size of that data. If not provided, update it since we know what it is.
	if val := methods["read"].tupleValue; val != nil && !e.Attributes.HasSize() {
		e.Attributes.SetSize(uint64(len(val.([]byte))))
	}

	entry := &pluginEntry{
		EntryBase:   plugin.NewEntry(e.Name),
		methods:     methods,
		state:       e.State,
		schemaKnown: schemaKnown,
		rawTypeID:   e.TypeID,
	}
	entry.SetAttributes(e.Attributes)
	entry.SetPartialMetadata(e.PartialMetadata)
	entry.setCacheTTLs(e.CacheTTLs)
	if e.InaccessibleReason != "" {
		entry.MarkInaccessible(ctx, fmt.Errorf(e.InaccessibleReason))
	}
	if e.SlashReplacer != "" {
		if len([]rune(e.SlashReplacer)) > 1 {
			msg := fmt.Sprintf("e.SlashReplacer: received string %v instead of a character", e.SlashReplacer)
			panic(msg)
		}

		entry.SetSlashReplacer([]rune(e.SlashReplacer)[0])
	}

	// If some data originated from the parent via list, mark as prefetched.
	if entry.methods["list"].tupleValue != nil || entry.methods["read"].tupleValue != nil {
		entry.Prefetched()
	}

	return entry, nil
}

// Adds a tupleValue to the signature. tupleValue is either a method-specific type, or
// json.RawMessage (so we can easily unmarshal it later).
type methodInfo struct {
	signature  plugin.MethodSignature
	tupleValue interface{}
}

// pluginEntry represents an external plugin entry
type pluginEntry struct {
	plugin.EntryBase
	script    pluginScript
	methods   map[string]methodInfo
	state     string
	rawTypeID string
	// schemaKnown is set by the root. We use it to enforce the invariant
	// "If the root implements schema, all entries must implement schema"
	// when decoding external plugin entries.
	schemaKnown bool
	// schemaGraphs is a map of <type_id> => <schema_graph>. It is created
	// by the root and passed along to child entries in list.
	schemaGraphs map[string]*linkedhashmap.Map
}

func (e *pluginEntry) setCacheTTLs(ttls decodedCacheTTLs) {
	if ttls.List != 0 {
		e.SetTTLOf(plugin.ListOp, ttls.List*time.Second)
	}
	if ttls.Read != 0 {
		e.SetTTLOf(plugin.ReadOp, ttls.Read*time.Second)
	}
	if ttls.Metadata != 0 {
		e.SetTTLOf(plugin.MetadataOp, ttls.Metadata*time.Second)
	}
}

// implements returns true if the entry implements the given method,
// false otherwise
func (e *pluginEntry) implements(method string) bool {
	_, ok := e.methods[method]
	return ok
}

func (e *pluginEntry) MethodSignature(name string) plugin.MethodSignature {
	if info, ok := e.methods[name]; ok {
		return info.signature
	}
	return plugin.UnsupportedSignature
}

func (e *pluginEntry) ChildSchemas() []*plugin.EntrySchema {
	// ChildSchema's meant for core plugins.
	return []*plugin.EntrySchema{}
}

func (e *pluginEntry) Schema() *plugin.EntrySchema {
	// This version of Schema's only meant for core plugins.
	return nil
}

func (e *pluginEntry) RawTypeID() string {
	return e.rawTypeID
}

const schemaFormat = `{
	"type_id_one":{
		"label": "one",
		"methods": ["list"],
		"children": ["type_id_two"]
	},
	"type_id_two":{
		"label":"two",
		"methods": ["read"]
	}
}
`

func rawTypeID(e *pluginEntry) string {
	rawTypeID := e.RawTypeID()
	if rawTypeID == "" {
		rawTypeID = "unknown"
	}
	return rawTypeID
}

func pluginName(e *pluginEntry) string {
	trimmedID := strings.Trim(e.ID(), "/")
	if trimmedID == "" {
		trimmedID = plugin.CName(e)
	}
	segments := strings.SplitN(trimmedID, "/", 2)
	return segments[0]
}

func namespace(pluginName string, rawTypeID string) string {
	return pluginName + "::" + rawTypeID
}

func (e *pluginEntry) SchemaGraph() (*linkedhashmap.Map, error) {
	if !e.implements("schema") {
		return nil, nil
	}
	var graph *linkedhashmap.Map
	if e.schemaGraphs != nil {
		g, ok := e.schemaGraphs[plugin.TypeID(e)]
		if !ok {
			msg := fmt.Errorf(
				"e.Schema(): entry schemas were prefetched, but no schema graph was provided for %v (%v)",
				plugin.ID(e),
				rawTypeID(e),
			)
			panic(msg)
		}
		graph = g
		// As a sanity check, ensure that the methods specified in the entry's schema
		// match the methods specified in the entry instance. Return an error if there
		// is a mismatch. Hopefully this should never happen.
		es, _ := graph.Get(plugin.TypeID(e))
		schemaMethods := es.(plugin.EntrySchema).Actions
		instanceMethods := []string{}
		for method := range e.methods {
			instanceMethods = append(instanceMethods, method)
		}
		sort.Strings(schemaMethods)
		sort.Strings(instanceMethods)
		mismatchErr := fmt.Errorf(
			"%v (%v): the schema's supported methods (%v) don't match the instance's supported methods (%v)",
			plugin.ID(e),
			rawTypeID(e),
			strings.Join(schemaMethods, ", "),
			strings.Join(instanceMethods, ", "),
		)
		if len(schemaMethods) != len(instanceMethods) {
			return nil, mismatchErr
		}
		for i := range instanceMethods {
			if instanceMethods[i] != schemaMethods[i] {
				return nil, mismatchErr
			}
		}
	} else {
		// Entry schemas were not prefetched, so we'll need to shell out. Even though entry
		// schemas should not change, shelling out is very useful for facilitating external
		// plugin development because it lets plugin authors see their schema changes live
		// without having to restart the Wash server.
		//
		// Entry schema generation should be fast, so pass-in a context w/ a 3 second timeout.
		ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelFunc()
		inv, err := e.script.InvokeAndWait(ctx, "schema", e)
		if err != nil {
			err := fmt.Errorf(
				"%v (%v): failed to retrieve the entry's schema: %v",
				plugin.ID(e),
				rawTypeID(e),
				err,
			)
			return nil, err
		}
		graph, err = unmarshalSchemaGraph(pluginName(e), rawTypeID(e), inv.Stdout().Bytes())
		if err != nil {
			err := fmt.Errorf(
				"%v (%v): could not decode schema from stdout: %v\nreceived:\n%v\nexpected something like:\n%v",
				plugin.ID(e),
				rawTypeID(e),
				err,
				strings.TrimSpace(inv.Stdout().String()),
				schemaFormat,
			)
			return nil, err
		}
	}
	return graph, nil
}

const listFormat = "[{\"name\":\"entry1\",\"methods\":[\"list\"]},{\"name\":\"entry2\",\"methods\":[\"list\"]}]"

func (e *pluginEntry) List(ctx context.Context) ([]plugin.Entry, error) {
	var decodedEntries []decodedExternalPluginEntry
	if impl := e.methods["list"].tupleValue; impl != nil {
		decodedEntries = impl.([]decodedExternalPluginEntry)
	} else {
		inv, err := e.script.InvokeAndWait(ctx, "list", e)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(inv.Stdout().Bytes(), &decodedEntries); err != nil {
			return nil, newStdoutDecodeErr(ctx, "the entries", err, inv, listFormat)
		}
	}

	entries := make([]plugin.Entry, len(decodedEntries))
	for i, decodedExternalPluginEntry := range decodedEntries {
		if coreEnt, ok := coreEntries[decodedExternalPluginEntry.TypeID]; ok {
			entry, err := coreEnt.createInstance(e, decodedExternalPluginEntry)
			if err != nil {
				return nil, err
			}
			entries[i] = entry
			continue
		}

		entry, err := decodedExternalPluginEntry.toExternalPluginEntry(ctx, e.schemaKnown, false)
		if err != nil {
			return nil, err
		}

		entry.script = e.script
		entry.schemaGraphs = e.schemaGraphs
		entries[i] = entry
	}

	return entries, nil
}

func (e *pluginEntry) Read(ctx context.Context) ([]byte, error) {
	if impl := e.methods["read"].tupleValue; impl != nil {
		return impl.([]byte), nil
	}

	inv, err := e.script.InvokeAndWait(ctx, "read", e)
	if err != nil {
		return nil, err
	}
	return inv.Stdout().Bytes(), nil
}

func (e *pluginEntry) BlockRead(ctx context.Context, size int64, offset int64) ([]byte, error) {
	inv, err := e.script.InvokeAndWait(ctx, "read", e, strconv.FormatInt(size, 10), strconv.FormatInt(offset, 10))
	if err != nil {
		return nil, err
	}
	return inv.Stdout().Bytes(), nil
}

func (e *pluginEntry) Write(ctx context.Context, p []byte) error {
	inv := e.script.NewInvocation(ctx, "write", e)
	inv.SetStdin(bytes.NewReader(p))
	return inv.RunAndWait(ctx)
}

func (e *pluginEntry) Metadata(ctx context.Context) (plugin.JSONObject, error) {
	if !e.implements("metadata") {
		// The entry does not override the "Metadata" method so invoke
		// the default
		return e.EntryBase.Metadata(ctx)
	}
	inv, err := e.script.InvokeAndWait(ctx, "metadata", e)
	if err != nil {
		return nil, err
	}
	var metadata plugin.JSONObject
	if err := json.Unmarshal(inv.Stdout().Bytes(), &metadata); err != nil {
		return nil, newStdoutDecodeErr(
			ctx,
			"the metadata",
			err,
			inv,
			"{\"key1\":\"value1\",\"key2\":\"value2\"}",
		)
	}
	return metadata, nil
}

func (e *pluginEntry) Signal(ctx context.Context, signal string) error {
	_, err := e.script.InvokeAndWait(ctx, "signal", e, signal)
	return err
}

func (e *pluginEntry) Delete(ctx context.Context) (deleted bool, err error) {
	inv, err := e.script.InvokeAndWait(ctx, "delete", e)
	if err != nil {
		return
	}
	if err = json.Unmarshal(inv.Stdout().Bytes(), &deleted); err != nil {
		err = newStdoutDecodeErr(
			ctx,
			"delete's result",
			err,
			inv,
			"true",
		)
		return
	}
	return
}

func (e *pluginEntry) Stream(ctx context.Context) (io.ReadCloser, error) {
	inv := e.script.NewInvocation(ctx, "stream", e)
	stdoutR, err := inv.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrR, err := inv.StderrPipe()
	if err != nil {
		return nil, err
	}
	activity.Record(ctx, "Starting %v", inv)
	if err := inv.Start(); err != nil {
		return nil, newInvokeError(err.Error(), inv)
	}
	// "wait" will be used in Stream's error handlers. It will be wrapped
	// in a "defer" call to ensure that cmd.Wait()'s called once we've read
	// all of stdout/stderr. These are the preconditions specified in
	// exec.Cmd#Wait's docs.
	wait := func() {
		if err := inv.Wait(); err != nil {
			activity.Record(ctx, "Failed waiting for %v to finish: %v", inv, err)
		}
	}

	// Wait for the header to appear on stdout. This lets us know that
	// the plugin's ready for streaming.
	header := "200"
	headerRdrCh := make(chan error, 1)
	go func() {
		headerBytes := []byte(header + "\n")
		buf := make([]byte, len(headerBytes), cap(headerBytes))
		n, err := stdoutR.Read(buf)
		if err != nil {
			headerRdrCh <- err
			return
		}
		if n != len(headerBytes) || string(buf) != string(headerBytes) {
			headerRdrCh <- fmt.Errorf("read an invalid header: %v", string(buf))
			return
		}
		headerRdrCh <- nil
	}()
	timeout := 5 * time.Second
	timer := time.After(timeout)
	select {
	case err := <-headerRdrCh:
		if err != nil {
			inv.Terminate()
			defer wait()
			// Try to get more context from stderr
			n, readErr := inv.Stderr().ReadFrom(stderrR)
			if readErr == nil && n > 0 {
				err = fmt.Errorf(inv.Stderr().String())
			}
			return nil, newInvokeError(fmt.Sprintf("failed to read the header: %v", err), inv)
		}
		// err == nil, meaning we've received the header. Keep reading from
		// stderr so that the streaming isn't blocked when its buffer is full.
		go func() {
			_, _ = io.Copy(ioutil.Discard, stderrR)
		}()
		return &stdoutStreamer{inv, stdoutR}, nil
	case <-timer:
		inv.Terminate()
		defer wait()
		// We timed out while waiting for the streaming header to appear.
		// Return an appropriate error message using whatever was printed
		// on stderr.
		errMsgFmt := fmt.Sprintf("did not see the %v header after %v seconds:", header, timeout)
		n, err := inv.Stderr().ReadFrom(stderrR)
		if err != nil {
			return nil, newInvokeError(fmt.Sprintf(
				errMsgFmt+" %v",
				fmt.Sprintf("cannot report reason: stderr i/o error: %v", err),
			), inv)
		}
		if n == 0 {
			return nil, newInvokeError(fmt.Sprintf(
				errMsgFmt+" %v",
				fmt.Sprintf("cannot report reason: nothing was printed to stderr"),
			), inv)
		}
		return nil, newInvokeError(errMsgFmt, inv)
	}
}

// Used for mocking tests.
var execSSHFn = transport.ExecSSH

func (e *pluginEntry) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	if result := e.methods["exec"].tupleValue; result != nil {
		impl := result.(execImpl)
		if impl.Transport != "ssh" {
			panic("transport must be ssh")
		}

		args = append([]string{cmd}, args...)
		return execSSHFn(ctx, impl.Options, args, opts)
	}

	// Serialize opts to JSON
	type serializedOptions struct {
		plugin.ExecOptions
		Stdin bool `json:"stdin"`
	}
	serializedOpts := serializedOptions{
		ExecOptions: opts,
		Stdin:       opts.Stdin != nil,
	}
	optsJSON, err := json.Marshal(serializedOpts)
	if err != nil {
		return nil, fmt.Errorf("could not marshal opts %v into JSON: %v", opts, err)
	}

	// Start the command.
	inv := e.script.NewInvocation(ctx, "exec", e, append([]string{string(optsJSON), cmd}, args...)...)
	execCmd := plugin.NewExecCommand(ctx)
	inv.SetStdout(execCmd.Stdout())
	inv.SetStderr(execCmd.Stderr())
	if opts.Stdin != nil {
		inv.SetStdin(opts.Stdin)
	} else {
		// Go's exec.Cmd reads from the null device if no stdin is provided. We instead provide
		// an empty string for input so plugins can test whether there is content to read.
		inv.SetStdin(strings.NewReader(""))
	}
	activity.Record(ctx, "Starting %v", inv)
	if err := inv.Start(); err != nil {
		return nil, err
	}
	// Command handles context-cancellation cleanup
	// for us, so we don't have to use execCmd.SetStopFunc.

	// Asynchronously wait for the command to finish
	go func() {
		err := inv.Wait()
		execCmd.CloseStreamsWithError(nil)
		exitCode := inv.ExitCode()
		if exitCode < 0 {
			execCmd.SetExitCodeErr(err)
		} else {
			execCmd.SetExitCode(exitCode)
		}
	}()
	return execCmd, nil
}

type stdoutStreamer struct {
	cmd    Command
	stdout io.ReadCloser
}

func (s *stdoutStreamer) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

func (s *stdoutStreamer) Close() error {
	s.cmd.Terminate()
	return s.cmd.Wait()
}

func newStdoutDecodeErr(ctx context.Context, decodedThing string, reason error, inv invocation, example string) error {
	activity.Record(
		ctx,
		"could not decode %v from stdout\nreceived:\n%v\nexpected something like:\n%v",
		decodedThing,
		strings.TrimSpace(inv.Stdout().String()),
		example,
	)
	return newInvokeError(fmt.Sprintf("could not decode %v from stdout: %v", decodedThing, reason), inv)
}

func unmarshalSchemaGraph(pluginName, rawTypeID string, stdout []byte) (*linkedhashmap.Map, error) {
	// Since we know e's type ID, it is OK if the serialized schema's keys are
	// out of order. However, the entry schema returned by the Wash API always
	// ensures that the first key is the entry's type ID. Thus, we unmarshal the
	// schema as a map[string]interface{} object, then convert it to a linkedhashmap
	// object so that we ensure the "first key == e.TypeID()" requirement of the API.
	// We'll also validate the unmarshalled schema in the latter conversion.
	var rawGraph map[string]interface{}
	if err := json.Unmarshal(stdout, &rawGraph); err != nil {
		return nil, fmt.Errorf("expected a non-empty JSON object")
	}
	if len(rawGraph) <= 0 {
		return nil, fmt.Errorf("expected a non-empty JSON object")
	}
	if rawGraph[rawTypeID] == nil {
		return nil, fmt.Errorf("%v's schema is missing", rawTypeID)
	}

	// Convert each node in the rawGraph to an EntrySchema object. This step
	// is also where we perform all of our validation. The validation consists
	// of ensuring that all required fields are present, and that the schema is
	// self-contained, i.e. that all child schemas are included. We will check
	// the latter condition after populating our graph via the
	// populatedTypeIDs/requiredTypeIDs variables.
	populatedTypeIDs := make(map[string]bool)
	requiredTypeIDs := map[string]bool{
		rawTypeID: true,
	}

	type decodedEntrySchema struct {
		plugin.EntrySchema
		Methods []string `json:"methods"`
	}

	graph := linkedhashmap.New()
	putNode := func(rawTypeID string, rawSchema interface{}) error {
		if coreEnt, ok := coreEntries[rawTypeID]; ok {
			pluginSchema := coreEnt.schema()

			// Copy only the public fields so we serialize it as just data. Uses copier because it uses
			// reflect to copy public fields, rather than Marshal/UnmarshalJSON which we've overridden.
			var schema plugin.EntrySchema
			err := copier.Copy(&schema, pluginSchema)
			if err != nil {
				panic(fmt.Sprintf("should always be able to copy from EntrySchema to EntrySchema: %v", err))
			}
			populatedTypeIDs[rawTypeID] = true
			graph.Put(namespace(pluginName, rawTypeID), schema)
			return nil
		}

		// Deep-copy the value into the decodedEntrySchema object
		schema, ok := rawSchema.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected a JSON object for %v's schema but got %v", rawTypeID, rawSchema)
		}
		var node decodedEntrySchema
		err := deepcopy.Copy(&node, schema)
		if err != nil {
			return err
		}

		// Ensure that all required fields are present
		populatedTypeIDs[rawTypeID] = true
		typeID := namespace(pluginName, rawTypeID)
		if len(node.Label) <= 0 {
			return fmt.Errorf("a label must be provided")
		}
		if node.Methods == nil {
			return fmt.Errorf("the entry's methods must be provided")
		}
		isParent := false
		isSignalable := false
		for _, method := range node.Methods {
			switch method {
			case "list":
				isParent = true
			case "signal":
				isSignalable = true
			}
		}
		if !isParent && len(node.Children) > 0 {
			return fmt.Errorf("entry has children even though it is not a parent. Parent entries must implement list")
		}
		if isParent {
			if len(node.Children) <= 0 {
				return fmt.Errorf("parent entries must specify their children's type IDs")
			}
			var namespacedChildren []string
			for _, child := range node.Children {
				requiredTypeIDs[child] = true
				namespacedChildren = append(namespacedChildren, namespace(pluginName, child))
			}
			node.Children = namespacedChildren
		}
		if !isSignalable && len(node.Signals) > 0 {
			return fmt.Errorf("entry has included a list of supported signals even though it is not signalable. Signalable entries must implement signal")
		}
		if isSignalable && len(node.Signals) <= 0 {
			return fmt.Errorf("signalable entries must include their supported signals")
		}
		if node.PartialMetadataSchema != nil && node.PartialMetadataSchema.Type.Type != "object" {
			return fmt.Errorf("invalid value for the partial metadata schema: expected a JSON object schema but got %v", node.PartialMetadataSchema.Type.Type)
		}
		if node.MetadataSchema != nil && node.MetadataSchema.Type.Type != "object" {
			return fmt.Errorf("invalid value for the metadata schema: expected a JSON object schema but got %v", node.MetadataSchema.Type.Type)
		}

		// All required fields are present, so put node.entrySchema in the graph.
		// We don't put node itself in because doing so would marshal its "Methods"
		// field.
		node.Actions = node.Methods
		graph.Put(typeID, node.EntrySchema)
		return nil
	}
	if err := putNode(rawTypeID, rawGraph[rawTypeID]); err != nil {
		return nil, err
	}
	delete(rawGraph, rawTypeID)
	for rawTypeID, schema := range rawGraph {
		if err := putNode(rawTypeID, schema); err != nil {
			return nil, err
		}
	}

	// Now validate that the schema's self-contained and that it does not
	// contain any dangling type IDs.
	for typeID := range requiredTypeIDs {
		if !populatedTypeIDs[typeID] {
			return nil, fmt.Errorf("%v's schema is missing", typeID)
		}
		delete(populatedTypeIDs, typeID)
	}
	if len(populatedTypeIDs) > 0 {
		var danglingTypeIDs []string
		for typeID := range populatedTypeIDs {
			danglingTypeIDs = append(danglingTypeIDs, typeID)
		}
		return nil, fmt.Errorf("the type IDs %v are not associated with any entry", strings.Join(danglingTypeIDs, ", "))
	}

	// Everything looks good, so return the graph
	return graph, nil
}

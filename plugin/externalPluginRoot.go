package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
)

// externalPluginRoot represents an external plugin's root.
type externalPluginRoot struct {
	*externalPluginEntry
}

// Init initializes the external plugin root
func (r *externalPluginRoot) Init(cfg map[string]interface{}) error {
	if cfg == nil {
		cfg = make(map[string]interface{})
	}
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal plugin config %v into JSON: %v", cfg, err)
	}

	// Give external plugins about five-seconds to finish their
	// initialization
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	inv, err := r.script.InvokeAndWait(ctx, "init", nil, string(cfgJSON))
	if err != nil {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out while waiting for init to finish")
		default:
			return err
		}
	}
	var decodedRoot decodedExternalPluginEntry
	if err := json.Unmarshal(inv.stdout.Bytes(), &decodedRoot); err != nil {
		return newStdoutDecodeErr(
			context.Background(),
			"the plugin root",
			err,
			inv,
			"{}",
		)
	}

	// Fill in required fields with data we already know.
	if decodedRoot.Name == "" {
		decodedRoot.Name = r.Name()
	} else if decodedRoot.Name != r.Name() {
		panic(fmt.Sprintf(`plugin root's name must match the basename (without extension) of %s
it's safe to omit name from the response to 'init'`, r.script.Path()))
	}
	if decodedRoot.Methods == nil {
		decodedRoot.Methods = []interface{}{"list"}
	}
	entry, err := decodedRoot.toExternalPluginEntry(false, true)
	if err != nil {
		return err
	}
	if !ListAction().IsSupportedOn(entry) {
		panic(fmt.Sprintf("plugin root for %s must implement 'list'", r.script.Path()))
	}
	script := r.script
	r.externalPluginEntry = entry
	r.externalPluginEntry.script = script

	// Fill in the schema graph if provided
	if rawSchema := r.methods["schema"]; rawSchema != nil {
		marshalledSchema, err := json.Marshal(rawSchema)
		if err != nil {
			panic(fmt.Sprintf("Error remarshaling previously unmarshaled data: %v", err))
		}
		graph, err := r.unmarshalSchemaGraph(marshalledSchema)
		if err != nil {
			return fmt.Errorf(
				"could not decode schema from stdout: %v\nreceived:\n%v\nexpected something like:\n%v",
				err,
				strings.TrimSpace(string(marshalledSchema)),
				schemaFormat,
			)
		}
		r.schemaGraphs = partitionSchemaGraph(graph)
	}

	return nil
}

func (r *externalPluginRoot) WrappedTypes() SchemaMap {
	// This only makes sense for core plugins because it is a Go-specific
	// limitation.
	return nil
}

// partitionSchemaGraph partitions graph into a map of <type_id> => <schema_graph>
func partitionSchemaGraph(graph *linkedhashmap.Map) map[string]*linkedhashmap.Map {
	var populate func(*linkedhashmap.Map, entrySchema, map[string]bool)
	populate = func(g *linkedhashmap.Map, node entrySchema, visited map[string]bool) {
		if visited[node.TypeID] {
			return
		}
		g.Put(node.TypeID, node)
		visited[node.TypeID] = true
		for _, childTypeID := range node.Children {
			childNode, ok := graph.Get(childTypeID)
			if !ok {
				msg := fmt.Sprintf("plugin.partitionSchemaGraph: expected child %v to be present in the graph", childTypeID)
				panic(msg)
			}
			populate(g, childNode.(entrySchema), visited)
		}
	}

	schemaGraphs := make(map[string]*linkedhashmap.Map)
	graph.Each(func(key interface{}, value interface{}) {
		g := linkedhashmap.New()
		populate(g, value.(entrySchema), make(map[string]bool))
		schemaGraphs[key.(string)] = g
	})

	return schemaGraphs
}

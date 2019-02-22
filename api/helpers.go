package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/plugin"
)

func findEntry(ctx context.Context, root plugin.Entry, segments []string) (plugin.Entry, *errorResponse) {
	path := strings.Join(segments, "/")

	curEntry := root
	curEntryID := "/" + root.Name()

	visitedSegments := make([]string, 0, cap(segments))
	for _, segment := range segments {
		switch curGroup := curEntry.(type) {
		case plugin.Group:
			// Get the entries via. LS()
			entries, err := plugin.CachedLS(ctx, curGroup, curEntryID)
			if err != nil {
				return nil, entryNotFoundResponse(path, err.Error())
			}

			// Search for the specific entry
			var found bool
			for _, entry := range entries {
				if entry.Name() == segment {
					found = true

					curEntry = entry
					curEntryID += "/" + segment
					visitedSegments = append(visitedSegments, segment)

					break
				}
			}
			if !found {
				reason := fmt.Sprintf("The %v entry does not exist", segment)
				if len(visitedSegments) != 0 {
					reason += fmt.Sprintf(" in the %v group", strings.Join(visitedSegments, "/"))
				}

				return nil, entryNotFoundResponse(path, reason)
			}
		default:
			reason := fmt.Sprintf("The entry %v is not a group", strings.Join(visitedSegments, "/"))
			return nil, entryNotFoundResponse(path, reason)
		}
	}

	return curEntry, nil
}

func getEntryFromPath(ctx context.Context, path string) (plugin.Entry, *errorResponse) {
	if path == "" {
		panic("path should never be empty")
	}

	// Don't interpret trailing slash as a new segment
	path = strings.TrimSuffix(path, "/")
	// Split into plugin name and an optional list of segments.
	segments := strings.Split(path, "/")
	pluginName := segments[0]
	segments = segments[1:]

	// Get the registry from context (added by registry middleware).
	registry := ctx.Value(pluginRegistryKey).(*plugin.Registry)
	root, ok := registry.Plugins[pluginName]
	if !ok {
		return nil, pluginDoesNotExistResponse(pluginName)
	}

	return findEntry(ctx, root, segments)
}

func toID(path string) string {
	return "/" + strings.TrimSuffix(path, "/")
}

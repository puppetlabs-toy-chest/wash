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
	visitedSegments := make([]string, 0, cap(segments))
	for _, segment := range segments {
		switch curGroup := curEntry.(type) {
		case plugin.Group:
			// Get the entries via. List()
			entries, err := plugin.CachedList(ctx, curGroup)
			if err != nil {
				if cnameErr, ok := err.(plugin.DuplicateCNameErr); ok {
					return nil, duplicateCNameResponse(cnameErr)
				}

				return nil, entryNotFoundResponse(path, err.Error())
			}

			// Search for the specific entry
			entry, ok := entries[segment]
			if !ok {
				reason := fmt.Sprintf("The %v entry does not exist", segment)
				if len(visitedSegments) != 0 {
					reason += fmt.Sprintf(" in the %v group", strings.Join(visitedSegments, "/"))
				}

				return nil, entryNotFoundResponse(path, reason)
			}

			curEntry = entry
			visitedSegments = append(visitedSegments, segment)
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

	// Don't interpret trailing slash as a new segment, and ignore optional leading slash
	path = strings.Trim(path, "/")
	// Split into plugin name and an optional list of segments.
	segments := strings.Split(path, "/")
	pluginName := segments[0]
	segments = segments[1:]

	// Get the registry from context (added by registry middleware).
	registry := ctx.Value(pluginRegistryKey).(*plugin.Registry)
	root, ok := registry.Plugins()[pluginName]
	if !ok {
		return nil, pluginDoesNotExistResponse(pluginName)
	}

	return findEntry(ctx, root, segments)
}

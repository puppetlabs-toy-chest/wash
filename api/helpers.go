package api

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
)

func toAPIEntry(e plugin.Entry) apitypes.Entry {
	return apitypes.Entry{
		Name:       plugin.Name(e),
		CName:      plugin.CName(e),
		Actions:    plugin.SupportedActionsOf(e),
		Attributes: plugin.Attributes(e),
	}
}

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
	if !filepath.IsAbs(path) {
		return nil, relativePathResponse(path)
	}

	mountpoint := ctx.Value(mountpointKey).(string)
	trimmedPath := strings.TrimPrefix(path, mountpoint)
	if trimmedPath == path {
		// TODO: Handle these later by wrapping them into entries
		return nil, nonWashEntryResponse(path)
	}
	path = trimmedPath

	// Get the registry from context (added by registry middleware).
	registry := ctx.Value(pluginRegistryKey).(*plugin.Registry)
	if path == "/" {
		// Return the registry
		return registry, nil
	}

	// Don't interpret trailing slash as a new segment, and ignore optional leading slash
	path = strings.Trim(path, "/")
	// Split into plugin name and an optional list of segments.
	segments := strings.Split(path, "/")
	pluginName := segments[0]
	segments = segments[1:]

	root, ok := registry.Plugins()[pluginName]
	if !ok {
		return nil, pluginDoesNotExistResponse(pluginName)
	}
	if len(segments) == 0 {
		// Listing the plugin itself, so return it's root
		return root, nil
	}

	return findEntry(ctx, root, segments)
}

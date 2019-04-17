package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	apifs "github.com/puppetlabs/wash/api/fs"
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

func getPathFromRequest(r *http.Request) (string, *errorResponse) {
	paths := r.URL.Query()["path"]
	if len(paths) != 1 {
		return "", invalidPathsResponse(paths)
	}
	return paths[0], nil
}

func getEntryFromRequest(ctx context.Context, r *http.Request) (plugin.Entry, string, *errorResponse) {
	path, errResp := getPathFromRequest(r)
	if errResp != nil {
		return nil, "", errResp
	}

	if path == "" {
		panic("path should never be empty")
	}
	if !filepath.IsAbs(path) {
		return nil, "", relativePathResponse(path)
	}

	mountpoint := ctx.Value(mountpointKey).(string)
	trimmedPath := strings.TrimPrefix(path, mountpoint)
	if trimmedPath == path {
		// Local file/directory, so convert it to a Wash entry
		//
		// TODO: The code here means that the Wash server cannot be
		// mounted on a remote machine. This is not an immediate issue,
		// but it does mean that we'll need to re-evaluate this code once
		// we get to the point where supporting remote Wash servers is
		// desirable.
		e, err := apifs.NewEntry(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, "", entryNotFoundResponse(path, err.Error())
			}
			err = fmt.Errorf("could not stat the regular file/dir pointed to by %v: %v", path, err)
			return nil, "", unknownErrorResponse(err)
		}
		return e, path, nil
	}
	// Don't interpret trailing slash as a new segment, and ignore optional leading slash
	trimmedPath = strings.Trim(trimmedPath, "/")

	// Get the registry from context (added by registry middleware).
	registry := ctx.Value(pluginRegistryKey).(*plugin.Registry)
	if trimmedPath == "" {
		// Return the registry
		return registry, path, nil
	}

	// Split into plugin name and an optional list of segments.
	segments := strings.Split(trimmedPath, "/")
	pluginName := segments[0]
	segments = segments[1:]

	root, ok := registry.Plugins()[pluginName]
	if !ok {
		return nil, "", pluginDoesNotExistResponse(pluginName)
	}
	if len(segments) == 0 {
		// Listing the plugin itself, so return it's root
		return root, path, nil
	}

	entry, err := findEntry(ctx, root, segments)
	return entry, path, err
}

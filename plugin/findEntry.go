package plugin

import (
	"context"
	"fmt"
	"strings"
)

// FindEntry returns the child of start found by following the segments, or an error if it cannot be found.
func FindEntry(ctx context.Context, start Entry, segments []string) (Entry, error) {
	visitedSegments := make([]string, 0, cap(segments))
	for _, segment := range segments {
		switch curParent := start.(type) {
		case Parent:
			// Get the entries via. List()
			entries, err := List(ctx, curParent)
			if err != nil {
				return nil, err
			}

			// Search for the specific entry
			entry, ok := entries[segment]
			if !ok {
				reason := fmt.Sprintf("The %v entry does not exist", segment)
				if len(visitedSegments) != 0 {
					reason += fmt.Sprintf(" in the %v parent", strings.Join(visitedSegments, "/"))
				}
				return nil, fmt.Errorf(reason)
			}

			start = entry
			visitedSegments = append(visitedSegments, segment)
		default:
			return nil, fmt.Errorf("The entry %v is not a parent", strings.Join(visitedSegments, "/"))
		}
	}

	return start, nil
}

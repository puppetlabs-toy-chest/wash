package volume

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/puppetlabs/wash/plugin"
)

func addAttributesForPath(dirmap DirMap, attr plugin.EntryAttributes, base, fullpath string, maxdepth int) {
	relative := strings.TrimPrefix(fullpath, base)
	// Create an entry for each directory.
	numSegments := numPathSegments(relative)
	if numSegments > maxdepth {
		panic(fmt.Sprintf("Should only have %v segments, found %v: %v", maxdepth, numSegments, relative))
	} else if attr.Mode().IsDir() {
		// Mark directories at maxdepth as unexplored with nil Children.
		if numSegments == maxdepth {
			dirmap[relative] = Children(nil)
		} else {
			dirmap[relative] = make(Children)
		}
	}

	// Add the entry to its parent's listing.
	parent, file := path.Split(relative)
	parent = strings.TrimSuffix(parent, "/")
	parentchildren := makeChildren(dirmap, parent)
	// Attr + path represents a volume dir or file.
	parentchildren[file] = attr
}

func numPathSegments(path string) int {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return 0
	}
	return len(strings.Split(path, string(os.PathSeparator)))
}

// Ensure the directory at newpath - and its parents - all exist. Returns the children of that
// directory. The directory may not exist if we're parsing stat output for a basepath that's closer
// to the root directory than where we searched because we want to preserve some of the hierarchy.
func makeChildren(dirmap DirMap, newpath string) Children {
	// If it exists, return the children map. Base case would be newpath == RootPath, which we create
	// at the start of ParseStatPOSIX.
	if newchildren, ok := dirmap[newpath]; ok {
		return newchildren
	}

	// Create the children for the new path.
	newchildren := make(Children)
	dirmap[newpath] = newchildren

	// Check if we need to create the parent, and get its children.
	parent, file := path.Split(newpath)
	parent = strings.TrimSuffix(parent, "/")
	parentchildren := makeChildren(dirmap, parent)

	// Add attributes for the new path to the parent's children. Then return the new map.
	attr := plugin.EntryAttributes{}
	attr.SetMode(os.ModeDir | 0550)
	parentchildren[file] = attr
	return newchildren
}

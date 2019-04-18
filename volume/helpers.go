// Package volume provides helpers for representing a remote filesystem.
//
// Plugins should use these helpers when representing a filesystem where the
// structure and stats are retrieved all-at-once. The filesystem representation
// should be stored in 'DirMap'. The root of the filesystem is then created with
// 'NewDir'.
package volume

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/puppetlabs/wash/munge"
	"github.com/puppetlabs/wash/plugin"
)

// StatCmd returns the command required to stat all the files in a directory.
func StatCmd(path string) []string {
	// size, atime, mtime, ctime, mode, name
	// %s - Total size, in bytes
	// %X - Time of last access as seconds since Epoch
	// %Y - Time of last data modification as seconds since Epoch
	// %Z - Time of last status change as seconds since Epoch
	// %f - Raw mode in hex
	// %n - File name
	return []string{"find", path, "-mindepth", "1", "-exec", "stat", "-c", "%s %X %Y %Z %f %n", "{}", "+"}
}

// TODO: Should move this over to plugin as a helper at some point
func parseTime(t string) (time.Time, error) {
	epoch, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(epoch, 0), nil
}

// StatParse parses a single line of the output of StatCmd into EntryAttributes and a name.
func StatParse(line string) (plugin.EntryAttributes, string, error) {
	var attr plugin.EntryAttributes
	segments := strings.SplitN(line, " ", 6)
	if len(segments) != 6 {
		return attr, "", fmt.Errorf("Stat did not return 6 components: %v", line)
	}

	size, err := strconv.ParseUint(segments[0], 10, 64)
	if err != nil {
		return attr, "", err
	}
	attr.SetSize(size)

	atime, err := parseTime(segments[1])
	if err != nil {
		return attr, "", err
	}
	attr.SetAtime(atime)

	mtime, err := parseTime(segments[2])
	if err != nil {
		return attr, "", err
	}
	attr.SetMtime(mtime)

	ctime, err := parseTime(segments[3])
	if err != nil {
		return attr, "", err
	}
	attr.SetCtime(ctime)

	mode, err := munge.ToFileMode("0x" + segments[4])
	if err != nil {
		return attr, "", err
	}
	attr.SetMode(mode)

	return attr, segments[5], nil
}

// A DirMap is a map of directory names to a map of their children and the children's attributes.
type DirMap = map[string]map[string]plugin.EntryAttributes

// StatParseAll an output stream that is the result of running StatCmd. Strips 'base' from the
// file paths, and maps each directory to a map of files in that directory and their attr
// (attributes).
func StatParseAll(output io.Reader, base string) (DirMap, error) {
	scanner := bufio.NewScanner(output)
	// Create lookup table for directories to contents, and prepopulate the root entry because
	// the mount point won't be included in the stat output.
	attrs := DirMap{"": make(map[string]plugin.EntryAttributes)}
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			attr, fullpath, err := StatParse(text)
			if err != nil {
				return nil, err
			}

			relative := strings.TrimPrefix(fullpath, base)
			// Create an entry for each directory.
			if attr.Mode().IsDir() {
				attrs[relative] = make(map[string]plugin.EntryAttributes)
			}

			// Add each entry to its parent's listing.
			parent, file := path.Split(relative)
			parent = strings.TrimSuffix(parent, "/")
			attrs[parent][file] = attr
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return attrs, nil
}

// MakeEntries constructs an array of entries for the given path from a DirMap.
// The root path is an empty string.
func MakeEntries(ctx context.Context, cacheEntry plugin.Entry, path string, listcb ListCB, contentcb ContentCB) ([]plugin.Entry, error) {
	result, err := plugin.CachedOp(ctx, "VolumeListCB", cacheEntry, 30*time.Second, func() (interface{}, error) {
		return listcb(ctx)
	})
	if err != nil {
		return nil, err
	}

	root := result.(DirMap)[path]
	entries := make([]plugin.Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode().IsDir() {
			entries = append(entries, NewDir(name, attr, cacheEntry, listcb, contentcb, path+"/"+name))
		} else {
			entries = append(entries, NewFile(name, attr, contentcb, path+"/"+name))
		}
	}
	return entries, nil
}

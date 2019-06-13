package volume

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/puppetlabs/wash/munge"
	"github.com/puppetlabs/wash/plugin"
)

// StatCmd returns the command required to stat all the files in a directory up to maxdepth.
func StatCmd(path string, maxdepth int) []string {
	// List uses "" to mean root. Translate for executing on the target.
	if path == "" {
		path = "/"
	}
	// size, atime, mtime, ctime, mode, name
	// %s - Total size, in bytes
	// %X - Time of last access as seconds since Epoch
	// %Y - Time of last data modification as seconds since Epoch
	// %Z - Time of last status change as seconds since Epoch
	// %f - Raw mode in hex
	// %n - File name
	return []string{"find", path, "-mindepth", "1", "-maxdepth", strconv.Itoa(maxdepth),
		"-exec", "stat", "-c", "%s %X %Y %Z %f %n", "{}", "+"}
}

// Keep as its own specialized function as it will be faster than munge.ToTime.
func parseTime(t string) (time.Time, error) {
	epoch, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(epoch, 0), nil
}

// StatParse parses a single line of the output of StatCmd into EntryAttributes and a path.
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

// Ensure the directory at newpath - and its parents - all exist. Return the contents of that
// directory. The directory may not exist if we're parsing stat output for a basepath that's closer
// to the root directory than where we searched because we want to preserve some of the hierarchy.
func makedirs(dirmap DirMap, newpath string) Dir {
	// If it exists, return the children map. Base case would be newpath == "", which we create at
	// the start of StatParseAll.
	if newchildren, ok := dirmap[newpath]; ok {
		return newchildren
	}

	// Create the attributes map for the new path.
	newchildren := make(Dir)
	dirmap[newpath] = newchildren

	// Check if we need to create the parent, and get its attributes map.
	parent, file := path.Split(newpath)
	parent = strings.TrimSuffix(parent, "/")
	parentchildren := makedirs(dirmap, parent)

	// Add attributes for the new path to the parent's attributes map. Then return the new map.
	attr := plugin.EntryAttributes{}
	attr.SetMode(os.ModeDir | 0550)
	parentchildren[file] = attr
	return newchildren
}

// StatParseAll an output stream that is the result of running StatCmd. Strips 'base' from the
// file paths, and maps each directory to a map of files in that directory and their attr
// (attributes). The 'maxdepth' used to produce the output is required to identify directories
// where we do not know their contents. 'start' denotes where the search started from, and is the
// basis for calculating maxdepth.
func StatParseAll(output io.Reader, base string, start string, maxdepth int) (DirMap, error) {
	maxdepth += numPathSegments(strings.TrimPrefix(start, base))
	scanner := bufio.NewScanner(output)
	// Create lookup table for directories to contents, and prepopulate the root entry because
	// the mount point won't be included in the stat output.
	dirmap := DirMap{"": make(Dir)}
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			attr, fullpath, err := StatParse(text)
			if err != nil {
				return nil, err
			}

			relative := strings.TrimPrefix(fullpath, base)
			// Create an entry for each directory.
			numSegments := numPathSegments(relative)
			if numSegments > maxdepth {
				panic(fmt.Sprintf("Should only have %v segments, found %v: %v", maxdepth, numSegments, relative))
			} else if attr.Mode().IsDir() {
				// Mark directories at maxdepth as unexplored with a nil Dir.
				if numSegments == maxdepth {
					dirmap[relative] = Dir(nil)
				} else {
					dirmap[relative] = make(Dir)
				}
			}

			// Add each entry to its parent's listing.
			parent, file := path.Split(relative)
			parent = strings.TrimSuffix(parent, "/")
			parentchildren := makedirs(dirmap, parent)
			// Attr + path represents a volume dir or file.
			parentchildren[file] = attr
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return dirmap, nil
}

func numPathSegments(path string) int {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return 0
	}
	return len(strings.Split(path, string(os.PathSeparator)))
}

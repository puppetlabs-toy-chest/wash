package volume

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

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
	return []string{"find", path, "-mindepth", "1", "-exec", "stat", "-c", "%s %X %Y %Z %f %n", "{}", ";"}
}

// StatParse parses a single line of the output of StatCmd into Attributes and a name.
func StatParse(line string) (plugin.Attributes, string, error) {
	var attr plugin.Attributes
	segments := strings.SplitN(line, " ", 6)
	if len(segments) != 6 {
		return attr, "", fmt.Errorf("Stat did not return 6 components: %v", line)
	}

	var err error
	attr.Size, err = strconv.ParseUint(segments[0], 10, 64)
	if err != nil {
		return attr, "", err
	}

	for i, target := range []*time.Time{&attr.Atime, &attr.Mtime, &attr.Ctime} {
		epoch, err := strconv.ParseInt(segments[i+1], 10, 64)
		if err != nil {
			return attr, "", err
		}
		*target = time.Unix(epoch, 0)
	}

	mode, err := strconv.ParseUint(segments[4], 16, 32)
	if err != nil {
		return attr, "", err
	}
	attr.Mode = plugin.ToFileMode(mode)

	return attr, segments[5], nil
}

// A DirMap is a map of directory names to a map of their children and the children's attributes.
type DirMap = map[string]map[string]plugin.Attributes

// StatParseAll an output stream that is the result of running StatCmd. Strips 'base' from the
// file paths, and maps each directory to a map of files in that directory and their attributes.
func StatParseAll(output io.Reader, base string) (DirMap, error) {
	scanner := bufio.NewScanner(output)
	// Create lookup table for directories to contents, and prepopulate the root entry because
	// the mount point won't be included in the stat output.
	attrs := DirMap{"": make(map[string]plugin.Attributes)}
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			attr, fullpath, err := StatParse(text)
			if err != nil {
				return nil, err
			}

			relative := strings.TrimPrefix(fullpath, base)
			// Create an entry for each directory.
			if attr.Mode.IsDir() {
				attrs[relative] = make(map[string]plugin.Attributes)
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

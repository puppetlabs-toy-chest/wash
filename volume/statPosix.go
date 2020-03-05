package volume

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/puppetlabs/wash/munge"
	"github.com/puppetlabs/wash/plugin"
)

// StatCmdPOSIX returns the POSIX command required to stat all the files in a directory up to maxdepth.
func StatCmdPOSIX(path string, maxdepth int) []string {
	// List uses "" to mean root. Translate for executing on the target.
	if path == RootPath {
		path = "/"
	}
	// size, atime, mtime, ctime, mode, name
	// %s - Total size, in bytes
	// %X - Time of last access as seconds since Epoch
	// %Y - Time of last data modification as seconds since Epoch
	// %Z - Time of last status change as seconds since Epoch
	// %f - Raw mode in hex
	// %n - File name
	// TODO: fix as part of https://github.com/puppetlabs/wash/issues/378. We don't currently handle
	// showing symbolic links, instead representing them as the resolved target.
	return []string{"find", "-L", path, "-mindepth", "1", "-maxdepth", strconv.Itoa(maxdepth),
		"-exec", "stat", "-L", "-c", "%s %X %Y %Z %f %n", "{}", "+"}
}

// Keep as its own specialized function as it will be faster than munge.ToTime.
func parseTime(t string) (time.Time, error) {
	epoch, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(epoch, 0), nil
}

// StatParse parses a single line of the output of StatCmdPOSIX into EntryAttributes and a path.
func parseStatPOSIX(line string) (plugin.EntryAttributes, string, error) {
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

// ParseStatPOSIX an output stream that is the result of running StatCmdPOSIX. Strips 'base' from the
// file paths, and maps each directory to a map of files in that directory and their attr
// (attributes). The 'maxdepth' used to produce the output is required to identify directories
// where we do not know their contents. 'start' denotes where the search started from, and is the
// basis for calculating maxdepth.
func ParseStatPOSIX(output io.Reader, base string, start string, maxdepth int) (DirMap, error) {
	maxdepth += numPathSegments(strings.TrimPrefix(start, base))
	scanner := bufio.NewScanner(output)
	// Create lookup table for directories to contents, and prepopulate the root entry because
	// the mount point won't be included in the stat output.
	dirmap := DirMap{RootPath: make(Children)}
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		// Skip error lines in case we're running in a tty.
		if text != "" && !NormalErrorPOSIX(text) {
			attr, fullpath, err := parseStatPOSIX(text)
			if err != nil {
				return nil, err
			}
			addAttributesForPath(dirmap, attr, base, fullpath, maxdepth)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return dirmap, nil
}

// NormalErrorPOSIX returns whether this line of text is normal error output for StatCmdPOSIX.
//
// Find may return a non-zero exit code, with messages in stdout like
//   stat: ‘/dev/fd/4’: No such file or directory
//   find: File system loop detected; ‘/proc/7/cwd’ is part of the same file system loop as ‘/’.
//   find: ‘/root’: Permission denied
// These are considered normal and handled by ParseStatPOSIX.
func NormalErrorPOSIX(text string) bool {
	return strings.HasPrefix(text, "stat:") || strings.HasPrefix(text, "find:")
}

package plugin

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
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

// StatParse parses a single line of the output of StatCmd into Attrbutes and a name.
func StatParse(line string) (Attributes, string, error) {
	var attr Attributes
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
	// mode output of stat is not directly convertible to os.FileMode.
	attr.Mode = os.FileMode(mode & 0777)
	for bits, mod := range map[uint64]os.FileMode{
		0140000: os.ModeSocket,
		0120000: os.ModeSymlink,
		// Skip file, absence of these implies a regular file.
		0060000: os.ModeDevice,
		0040000: os.ModeDir,
		0020000: os.ModeCharDevice,
		0010000: os.ModeNamedPipe,
		0004000: os.ModeSetuid,
		0002000: os.ModeSetgid,
		0001000: os.ModeSticky,
	} {
		if mode&bits != 0 {
			attr.Mode |= mod
		}
	}

	return attr, segments[5], nil
}

// StatParseAll an output stream that is the result of running StatCmd.
// Strips 'base' from the file paths, and maps each directory (full path with trailing slash)
// to a map of files in that directory and their attributes.
func StatParseAll(output io.Reader, base string) (map[string]map[string]Attributes, error) {
	scanner := bufio.NewScanner(output)
	attrs := make(map[string]map[string]Attributes)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			attr, fullpath, err := StatParse(text)
			if err != nil {
				return nil, err
			}

			relative := strings.TrimPrefix(fullpath, base)
			parent, file := path.Split(relative)
			if len(attrs[parent]) == 0 {
				attrs[parent] = make(map[string]Attributes)
			}
			attrs[parent][file] = attr
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return attrs, nil
}

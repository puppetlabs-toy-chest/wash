package volume

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// ChildItemsCmd returns the PowerShell command required to stat all the files in a directory up to maxdepth.
func ChildItemsCmd(path string, maxdepth int) []string {
	// List uses "" to mean root. Translate for executing on the target.
	if path == RootPath {
		path = "/"
	}
	// Execute as a single command because it's a powershell expression, not an executable and argument list.
	// Get-ChildItem implicitly includes one level, -Depth is additional levels to recurse. Subtract
	// one from maxdepth to account for this.
	// TODO: fix as part of https://github.com/puppetlabs/wash/issues/378. We don't currently handle
	// showing symbolic links, instead representing them as the resolved target. (Target,LinkType?)
	return []string{
		"Get-ChildItem '" + path + "' -Recurse -Depth " + strconv.Itoa(maxdepth-1) +
			" | Select-Object FullName,Length,CreationTimeUtc,LastAccessTimeUtc,LastWriteTimeUtc,Attributes" +
			` | ForEach-Object {
$utc=[Xml.XmlDateTimeSerializationMode]::Utc
$_.CreationTimeUtc = [Xml.XmlConvert]::ToString($_.CreationTimeUtc,$utc)
$_.LastAccessTimeUtc = [Xml.XmlConvert]::ToString($_.LastAccessTimeUtc,$utc)
$_.LastWriteTimeUtc = [Xml.XmlConvert]::ToString($_.LastWriteTimeUtc,$utc)
$_ } | ConvertTo-Csv`,
	}
}

// parseItem parses a single csv record of ChildItemsCmd into EntryAttributes and a path.
// Example: "C:\Windows\WindowsUpdate.log","276","10/13/2019 1:22:54 AM","10/13/2019 1:28:08 AM","10/13/2019 1:28:08 AM","Archive, Compressed"
func parseItem(record []string) (attr plugin.EntryAttributes, path string, err error) {
	// Normalize path to remove the drive letter and convert to forward-slash
	path = strings.TrimLeftFunc(record[0], func(c rune) bool { return c != '\\' })
	path = strings.ReplaceAll(path, "\\", "/")

	if record[1] != "" {
		var length uint64
		length, err = strconv.ParseUint(record[1], 10, 64)
		if err != nil {
			return
		}
		attr.SetSize(length)
	} else {
		attr.SetSize(0)
	}

	var crtime, atime, mtime time.Time
	if crtime, err = time.Parse(time.RFC3339, record[2]); err != nil {
		return
	}
	attr.SetCrtime(crtime)
	if atime, err = time.Parse(time.RFC3339, record[3]); err != nil {
		return
	}
	attr.SetAtime(atime)
	if mtime, err = time.Parse(time.RFC3339, record[4]); err != nil {
		return
	}
	attr.SetMtime(mtime)
	// modified implies changed, so this seems appropriate
	attr.SetCtime(mtime)

	// Attributes: Archive, Compressed, Device, Directory, Encrypted, Hidden, IntegrityStream, Normal,
	//						 NoScrubData, NotContentIndexed, Offline, ReadOnly, ReparsePoint, SparseFile, System, Temporary
	var mode os.FileMode = 0600
	attributes := strings.Split(record[5], ", ")
	for _, a := range attributes {
		switch a {
		case "Directory":
			mode |= os.ModeDir
		case "ReadOnly":
			mode &^= 0200 // Bitclear
		}
	}
	attr.SetMode(mode)
	return
}

// ItemsParseAll an output stream that is the result of running ChildItemsCmd. Strips 'base' from the
// file paths, and maps each directory to a map of files in that directory and their attr
// (attributes). The 'maxdepth' used to produce the output is required to identify directories
// where we do not know their contents. 'start' denotes where the search started from, and is the
// basis for calculating maxdepth.
func ItemsParseAll(output io.Reader, base string, start string, maxdepth int) (DirMap, error) {
	maxdepth += numPathSegments(strings.TrimPrefix(start, base))
	scanner := csv.NewReader(output)
	scanner.Comment = '#'
	scanner.FieldsPerRecord = 6
	records, err := scanner.ReadAll()
	if err != nil {
		return nil, err
	}

	// Create lookup table for directories to contents, and prepopulate the root entry because
	// the mount point won't be included in the stat output.
	dirmap := DirMap{RootPath: make(Children)}
	if len(records) == 0 {
		// Start directory was empty.
		return dirmap, nil
	}

	// Skip header field
	for _, record := range records[1:] {
		attr, fullpath, err := parseItem(record)
		if err != nil {
			return nil, err
		}
		addAttributesForPath(dirmap, attr, base, fullpath, maxdepth)
	}
	return dirmap, nil
}

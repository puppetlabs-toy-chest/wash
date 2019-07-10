package volume

import (
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

// Generated with
// `docker run --rm -it -v=/test/fixture:/mnt busybox find /mnt/ -mindepth 1 -exec stat -c '%s %X %Y %Z %f %n' {} \;`
const mountpoint = "mnt"
const mountDepth = 5
const fixture = `
96 1550611510 1550611448 1550611448 41ed mnt/path
96 1550611510 1550611448 1550611448 41ed mnt/path/has
96 1550611510 1550611448 1550611448 41ed mnt/path/has/got
96 1550611510 1550611458 1550611458 41ed mnt/path/has/got/some
0 1550611458 1550611458 1550611458 81a4 mnt/path/has/got/some/legs
96 1550611510 1550611453 1550611453 41ed mnt/path1
0 1550611453 1550611453 1550611453 81a4 mnt/path1/a file
96 1550611510 1550611441 1550611441 41ed mnt/path2
64 1550611510 1550611441 1550611441 41ed mnt/path2/dir
`

func TestStatCmd(t *testing.T) {
	cmd := StatCmd("", 1)
	assert.Equal(t, []string{"find", "-L", "/", "-mindepth", "1", "-maxdepth", "1",
		"-exec", "stat", "-L", "-c", "%s %X %Y %Z %f %n", "{}", "+"}, cmd)

	cmd = StatCmd("/", 1)
	assert.Equal(t, []string{"find", "-L", "/", "-mindepth", "1", "-maxdepth", "1",
		"-exec", "stat", "-L", "-c", "%s %X %Y %Z %f %n", "{}", "+"}, cmd)

	cmd = StatCmd("/var/log", 5)
	assert.Equal(t, []string{"find", "-L", "/var/log", "-mindepth", "1", "-maxdepth", "5",
		"-exec", "stat", "-L", "-c", "%s %X %Y %Z %f %n", "{}", "+"}, cmd)
}

func TestStatParse(t *testing.T) {
	actualAttr, path, err := StatParse("96 1550611510 1550611448 1550611448 41ed mnt/path")
	assert.Nil(t, err)
	assert.Equal(t, "mnt/path", path)
	expectedAttr := plugin.EntryAttributes{}
	expectedAttr.
		SetAtime(time.Unix(1550611510, 0)).
		SetMtime(time.Unix(1550611448, 0)).
		SetCtime(time.Unix(1550611448, 0)).
		SetMode(0755 | os.ModeDir).
		SetSize(96)
	assert.Equal(t, expectedAttr, actualAttr)

	actualAttr, path, err = StatParse("0 1550611458 1550611458 1550611458 81a4 mnt/path/has/got/some/legs")
	assert.Nil(t, err)
	assert.Equal(t, "mnt/path/has/got/some/legs", path)
	expectedAttr = plugin.EntryAttributes{}
	expectedAttr.
		SetAtime(time.Unix(1550611458, 0)).
		SetMtime(time.Unix(1550611458, 0)).
		SetCtime(time.Unix(1550611458, 0)).
		SetMode(0644).
		SetSize(0)
	assert.Equal(t, expectedAttr, actualAttr)

	_, _, err = StatParse("stat: failed")
	assert.Equal(t, errors.New("Stat did not return 6 components: stat: failed"), err)

	_, _, err = StatParse("-1 1550611510 1550611448 1550611448 41ed mnt/path")
	if assert.NotNil(t, err) {
		assert.Equal(t, &strconv.NumError{Func: "ParseUint", Num: "-1", Err: strconv.ErrSyntax}, err)
	}

	_, _, err = StatParse("0 2019-01-01 2019-01-01 2019-01-01 41ed mnt/path")
	if assert.NotNil(t, err) {
		assert.Equal(t, &strconv.NumError{Func: "ParseInt", Num: "2019-01-01", Err: strconv.ErrSyntax}, err)
	}

	_, _, err = StatParse("96 1550611510 1550611448 1550611448 zebra mnt/path")
	if assert.NotNil(t, err) {
		assert.Regexp(t, regexp.MustCompile("parse.*mode.*zebra"), err.Error())
	}
}

func TestStatParseAll(t *testing.T) {
	dmap, err := StatParseAll(strings.NewReader(fixture), mountpoint, mountpoint, mountDepth)
	assert.Nil(t, err)
	assert.NotNil(t, dmap)
	assert.Equal(t, 8, len(dmap))
	for _, dir := range []string{RootPath, "/path", "/path/has", "/path/has/got", "/path/has/got/some", "/path1", "/path2", "/path2/dir"} {
		assert.Contains(t, dmap, dir)
	}
	for _, file := range []string{"/path/has/got/some/legs", "/path1/a file"} {
		assert.NotContains(t, dmap, file)
	}

	for _, node := range []string{"path", "path1", "path2"} {
		assert.Contains(t, dmap[RootPath], node)
	}

	expectedAttr := plugin.EntryAttributes{}
	expectedAttr.
		SetAtime(time.Unix(1550611453, 0)).
		SetMtime(time.Unix(1550611453, 0)).
		SetCtime(time.Unix(1550611453, 0)).
		SetMode(0644).
		SetSize(0)
	assert.Equal(t, expectedAttr, dmap["/path1"]["a file"])

	expectedAttr = plugin.EntryAttributes{}
	expectedAttr.
		SetAtime(time.Unix(1550611510, 0)).
		SetMtime(time.Unix(1550611441, 0)).
		SetCtime(time.Unix(1550611441, 0)).
		SetMode(0755 | os.ModeDir).
		SetSize(64)
	assert.Equal(t, expectedAttr, dmap["/path2"]["dir"])

	expectedAttr = plugin.EntryAttributes{}
	expectedAttr.
		SetAtime(time.Unix(1550611510, 0)).
		SetMtime(time.Unix(1550611448, 0)).
		SetCtime(time.Unix(1550611448, 0)).
		SetMode(0755 | os.ModeDir).
		SetSize(96)
	assert.Equal(t, expectedAttr, dmap["/path"]["has"])
}

func TestStatParseAllUnfinished(t *testing.T) {
	const shortFixture = `
	96 1550611510 1550611448 1550611448 41ed mnt/path
	96 1550611510 1550611448 1550611448 41ed mnt/path/has
	`
	dmap, err := StatParseAll(strings.NewReader(shortFixture), mountpoint, mountpoint, 2)
	assert.Nil(t, err)
	assert.NotNil(t, dmap)
	assert.Equal(t, 3, len(dmap))
	assert.Contains(t, dmap, RootPath)
	assert.Contains(t, dmap[RootPath], "path")
	assert.Contains(t, dmap, "/path")
	assert.Contains(t, dmap["/path"], "has")
	// Depth two paths will be nil to signify we don't know their children.
	assert.Contains(t, dmap, "/path/has")
	assert.Nil(t, dmap["/path/has"])
}

func TestStatParseAllDeep(t *testing.T) {
	const shortFixture = `
	96 1550611510 1550611448 1550611448 41ed mnt/path
	96 1550611510 1550611448 1550611448 41ed mnt/path/has
	`
	dmap, err := StatParseAll(strings.NewReader(shortFixture), RootPath, mountpoint, 2)
	assert.Nil(t, err)
	assert.NotNil(t, dmap)
	assert.Equal(t, 4, len(dmap))
	assert.Contains(t, dmap, RootPath)
	assert.Contains(t, dmap[RootPath], "mnt")
	assert.Contains(t, dmap, "mnt")
	assert.Contains(t, dmap["mnt"], "path")
	assert.Contains(t, dmap, "mnt/path")
	assert.Contains(t, dmap["mnt/path"], "has")
	// Depth two paths will be nil to signify we don't know their children.
	assert.Contains(t, dmap, "mnt/path/has")
	assert.Nil(t, dmap["mnt/path/has"])
}

func TestStatParseAllRoot(t *testing.T) {
	dmap, err := StatParseAll(strings.NewReader(fixture), RootPath, RootPath, mountDepth+1)
	assert.Nil(t, err)
	assert.NotNil(t, dmap)
	assert.Equal(t, 9, len(dmap))
	for _, dir := range []string{RootPath, "mnt", "mnt/path", "mnt/path/has", "mnt/path/has/got", "mnt/path/has/got/some", "mnt/path1", "mnt/path2", "mnt/path2/dir"} {
		assert.Contains(t, dmap, dir)
	}
	for _, file := range []string{"mnt/path/has/got/some/legs", "mnt/path1/a file"} {
		assert.NotContains(t, dmap, file)
	}

	for _, node := range []string{"path", "path1", "path2"} {
		assert.Contains(t, dmap["mnt"], node)
	}

	expectedAttr := plugin.EntryAttributes{}
	expectedAttr.
		SetAtime(time.Unix(1550611453, 0)).
		SetMtime(time.Unix(1550611453, 0)).
		SetCtime(time.Unix(1550611453, 0)).
		SetMode(0644).
		SetSize(0)
	assert.Equal(t, expectedAttr, dmap["mnt/path1"]["a file"])

	expectedAttr = plugin.EntryAttributes{}
	expectedAttr.
		SetAtime(time.Unix(1550611510, 0)).
		SetMtime(time.Unix(1550611441, 0)).
		SetCtime(time.Unix(1550611441, 0)).
		SetMode(0755 | os.ModeDir).
		SetSize(64)
	assert.Equal(t, expectedAttr, dmap["mnt/path2"]["dir"])

	expectedAttr = plugin.EntryAttributes{}
	expectedAttr.
		SetAtime(time.Unix(1550611510, 0)).
		SetMtime(time.Unix(1550611448, 0)).
		SetCtime(time.Unix(1550611448, 0)).
		SetMode(0755 | os.ModeDir).
		SetSize(96)
	assert.Equal(t, expectedAttr, dmap["mnt/path"]["has"])

	expectedAttr = plugin.EntryAttributes{}
	expectedAttr.SetMode(0550 | os.ModeDir)
	assert.Equal(t, expectedAttr, dmap[RootPath]["mnt"])
}

func TestNumPathSegments(t *testing.T) {
	assert.Equal(t, 0, numPathSegments(""))
	assert.Equal(t, 0, numPathSegments("/"))
	assert.Equal(t, 1, numPathSegments("/foo"))
	assert.Equal(t, 2, numPathSegments("/foo/bar"))
}

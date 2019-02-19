package os

import (
	"errors"
	"os"
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

func TestStatParse(t *testing.T) {
	attr, path, err := StatParse("96 1550611510 1550611448 1550611448 41ed mnt/path")
	assert.Nil(t, err)
	assert.Equal(t, "mnt/path", path)
	assert.Equal(t, plugin.Attributes{
		Atime: time.Unix(1550611510, 0),
		Mtime: time.Unix(1550611448, 0),
		Ctime: time.Unix(1550611448, 0),
		Mode:  0755 | os.ModeDir,
		Size:  96,
	}, attr)

	attr, path, err = StatParse("0 1550611458 1550611458 1550611458 81a4 mnt/path/has/got/some/legs")
	assert.Nil(t, err)
	assert.Equal(t, "mnt/path/has/got/some/legs", path)
	assert.Equal(t, plugin.Attributes{
		Atime: time.Unix(1550611458, 0),
		Mtime: time.Unix(1550611458, 0),
		Ctime: time.Unix(1550611458, 0),
		Mode:  0644,
		Size:  0,
	}, attr)

	attr, path, err = StatParse("stat: failed")
	assert.Equal(t, errors.New("Stat did not return 6 components: stat: failed"), err)

	attr, path, err = StatParse("-1 1550611510 1550611448 1550611448 41ed mnt/path")
	if assert.NotNil(t, err) {
		assert.Equal(t, &strconv.NumError{Func: "ParseUint", Num: "-1", Err: strconv.ErrSyntax}, err)
	}

	attr, path, err = StatParse("0 2019-01-01 2019-01-01 2019-01-01 41ed mnt/path")
	if assert.NotNil(t, err) {
		assert.Equal(t, &strconv.NumError{Func: "ParseInt", Num: "2019-01-01", Err: strconv.ErrSyntax}, err)
	}

	attr, path, err = StatParse("96 1550611510 1550611448 1550611448 zebra mnt/path")
	if assert.NotNil(t, err) {
		assert.Equal(t, &strconv.NumError{Func: "ParseUint", Num: "zebra", Err: strconv.ErrSyntax}, err)
	}
}

func TestStatParseAll(t *testing.T) {
	dmap, err := StatParseAll(strings.NewReader(fixture), mountpoint)
	assert.Nil(t, err)
	assert.NotNil(t, dmap)
	assert.Equal(t, 8, len(dmap))
	for _, dir := range []string{"", "/path", "/path/has", "/path/has/got", "/path/has/got/some", "/path1", "/path2", "/path2/dir"} {
		assert.NotNil(t, dmap[dir])
	}
	for _, file := range []string{"/path/has/got/some/legs", "/path1/a file"} {
		assert.Nil(t, dmap[file])
	}

	for _, node := range []string{"/path", "/path1", "/path2"} {
		assert.NotNil(t, dmap[""][node])
	}

	assert.Equal(t, plugin.Attributes{
		Atime: time.Unix(1550611453, 0),
		Mtime: time.Unix(1550611453, 0),
		Ctime: time.Unix(1550611453, 0),
		Mode:  0644,
		Size:  0,
	}, dmap["/path1"]["a file"])

	assert.Equal(t, plugin.Attributes{
		Atime: time.Unix(1550611510, 0),
		Mtime: time.Unix(1550611441, 0),
		Ctime: time.Unix(1550611441, 0),
		Mode:  0755 | os.ModeDir,
		Size:  64,
	}, dmap["/path2"]["dir"])

	assert.Equal(t, plugin.Attributes{
		Atime: time.Unix(1550611510, 0),
		Mtime: time.Unix(1550611448, 0),
		Ctime: time.Unix(1550611448, 0),
		Mode:  0755 | os.ModeDir,
		Size:  96,
	}, dmap["/path"]["has"])
}

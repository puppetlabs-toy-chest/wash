package shell

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "testBasic")
	if !assert.NoError(t, err) {
		assert.FailNow(t, "Temporary directory required for further testing")
	}
	defer os.RemoveAll(tmpdir)

	sh := basic{sh: "/bin/sh"}
	_, err = sh.Command([]string{"help"}, tmpdir)
	assert.NoError(t, err)

	helpfile := filepath.Join(tmpdir, "help")
	assert.FileExists(t, helpfile)
	bits, err := ioutil.ReadFile(helpfile)
	assert.NoError(t, err)
	assert.Contains(t, string(bits), "#!/bin/sh\nWASH_EMBEDDED=1 exec wash help \"$@\"")
}

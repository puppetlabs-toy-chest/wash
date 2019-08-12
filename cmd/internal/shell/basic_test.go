package shell

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
	comm, err := sh.Command([]string{"help"}, tmpdir)
	assert.NoError(t, err)
	comm.Stdin = strings.NewReader(filepath.Join(tmpdir, "help"))
	output, err := comm.Output()
	assert.NoError(t, err)
	assert.Contains(t, string(output), "Available Commands")
}

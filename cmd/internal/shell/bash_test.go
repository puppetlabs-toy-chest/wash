package shell

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBash(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "testBash")
	if !assert.NoError(t, err) {
		assert.FailNow(t, "Temporary directory required for further testing")
	}
	defer os.RemoveAll(tmpdir)

	sh := bash{sh: "/bin/bash"}
	comm, err := sh.Command([]string{"help"}, tmpdir)
	assert.NoError(t, err)
	assert.NotEmpty(t, comm.Env)
	bashenv := filepath.Join(tmpdir, ".bashenv")
	assert.Contains(t, comm.Env, "BASH_ENV="+bashenv)

	assert.FileExists(t, bashenv)
	bits, err := ioutil.ReadFile(bashenv)
	assert.NoError(t, err)
	assert.Contains(t, string(bits), "alias help='WASH_EMBEDDED=1 wash help'")

	assert.FileExists(t, filepath.Join(tmpdir, ".bashrc"))
}

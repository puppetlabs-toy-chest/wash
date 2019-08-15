package shell

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZsh(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "testZsh")
	if !assert.NoError(t, err) {
		assert.FailNow(t, "Temporary directory required for further testing")
	}
	defer os.RemoveAll(tmpdir)

	sh := zsh{sh: "/bin/zsh"}
	comm, err := sh.Command([]string{"help"}, tmpdir)
	assert.NoError(t, err)
	assert.NotEmpty(t, comm.Env)
	assert.Contains(t, comm.Env, "ZDOTDIR="+tmpdir)

	zshenv := filepath.Join(tmpdir, ".zshenv")
	assert.FileExists(t, zshenv)
	bits, err := ioutil.ReadFile(zshenv)
	assert.NoError(t, err)
	assert.Contains(t, string(bits), "alias help='WASH_EMBEDDED=1 wash help'")

	assert.FileExists(t, filepath.Join(tmpdir, ".zshrc"))
}

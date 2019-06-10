package volume

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

type mockFileEntry struct {
	plugin.EntryBase
	content string
	err     error
}

func (m *mockFileEntry) VolumeList(context.Context) (DirMap, error) {
	return nil, nil
}

func (m *mockFileEntry) VolumeOpen(context.Context, string) (plugin.SizedReader, error) {
	if m.err != nil {
		return nil, m.err
	}
	return strings.NewReader(m.content), nil
}

func (m *mockFileEntry) VolumeStream(context.Context, string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	return ioutil.NopCloser(strings.NewReader(m.content)), nil
}

func TestVolumeFile(t *testing.T) {
	now := time.Now()
	initialAttr := plugin.EntryAttributes{}
	initialAttr.SetCtime(now)

	impl := &mockFileEntry{EntryBase: plugin.NewEntry(), content: "hello"}
	impl.SetName("parent")
	vf := newFile("mine", initialAttr, impl, "my path")

	attr := plugin.Attributes(vf)
	expectedAttr := plugin.EntryAttributes{}
	expectedAttr.SetCtime(now)
	assert.Equal(t, expectedAttr, attr)

	rdr, err := vf.Open(context.Background())
	assert.Nil(t, err)
	if assert.NotNil(t, rdr) {
		buf := make([]byte, rdr.Size())
		n, err := rdr.ReadAt(buf, 0)
		assert.Nil(t, err)
		assert.Equal(t, int64(n), rdr.Size())
		assert.Equal(t, "hello", string(buf))
	}

	rdr2, err := vf.Stream(context.Background())
	assert.Nil(t, err)
	if assert.NotNil(t, rdr2) {
		buf, err := ioutil.ReadAll(rdr2)
		if assert.NoError(t, err) {
			assert.Equal(t, "hello", string(buf))
		}
	}
}

func TestVolumeFileErr(t *testing.T) {
	impl := &mockFileEntry{EntryBase: plugin.NewEntry(), err: errors.New("fail")}
	impl.SetName("parent")
	vf := newFile("mine", plugin.EntryAttributes{}, impl, "my path")

	rdr, err := vf.Open(context.Background())
	assert.Nil(t, rdr)
	assert.Equal(t, errors.New("fail"), err)

	rdr2, err := vf.Stream(context.Background())
	assert.Nil(t, rdr2)
	assert.Equal(t, errors.New("fail"), err)
}

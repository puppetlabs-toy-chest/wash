package volume

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
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

func (m *mockFileEntry) VolumeList(context.Context, string) (DirMap, error) {
	return nil, nil
}

func (m *mockFileEntry) VolumeRead(context.Context, string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []byte(m.content), nil
}

func (m *mockFileEntry) VolumeStream(context.Context, string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	return ioutil.NopCloser(strings.NewReader(m.content)), nil
}

func (m *mockFileEntry) VolumeWrite(_ context.Context, _ string, b []byte, _ os.FileMode) error {
	if m.err != nil {
		return m.err
	}
	m.content = string(b)
	return nil
}

func (m *mockFileEntry) VolumeDelete(context.Context, string) (bool, error) {
	return true, nil
}

func (m *mockFileEntry) Schema() *plugin.EntrySchema {
	return nil
}

func TestVolumeFile(t *testing.T) {
	now := time.Now()
	initialAttr := plugin.EntryAttributes{}
	initialAttr.SetCtime(now)

	impl := &mockFileEntry{EntryBase: plugin.NewEntry("parent"), content: "hello"}
	vf := newFile("mine", initialAttr, impl, "my path")

	attr := plugin.Attributes(vf)
	expectedAttr := plugin.EntryAttributes{}
	expectedAttr.SetCtime(now)
	assert.Equal(t, expectedAttr, attr)

	content, err := vf.Read(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, []byte("hello"), content)

	rdr, err := plugin.Stream(context.Background(), vf)
	assert.Nil(t, err)
	if assert.NotNil(t, rdr) {
		buf, err := ioutil.ReadAll(rdr)
		if assert.NoError(t, err) {
			assert.Equal(t, "hello", string(buf))
		}
	}

	text := "some text"
	err = vf.Write(context.Background(), []byte(text))
	assert.NoError(t, err)
	assert.Equal(t, text, impl.content)
}

func TestVolumeFileErr(t *testing.T) {
	impl := &mockFileEntry{EntryBase: plugin.NewEntry("parent"), err: errors.New("fail")}
	vf := newFile("mine", plugin.EntryAttributes{}, impl, "my path")

	content, err := vf.Read(context.Background())
	assert.Nil(t, content)
	assert.Equal(t, errors.New("fail"), err)

	rdr, err := plugin.Stream(context.Background(), vf)
	assert.Nil(t, rdr)
	assert.Equal(t, errors.New("fail"), err)

	err = vf.Write(context.Background(), []byte{'a'})
	assert.Equal(t, errors.New("fail"), err)
}

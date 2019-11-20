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

func (m *mockFileEntry) VolumeList(context.Context, string) (DirMap, error) {
	return nil, nil
}

func (m *mockFileEntry) VolumeRead(context.Context, string) (io.ReaderAt, error) {
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

	buf := make([]byte, 5)
	n, err := vf.Read(context.Background(), buf, 0)
	if assert.Nil(t, err) {
		assert.Equal(t, 5, n)
		assert.Equal(t, "hello", string(buf))
	}

	rdr2, err := plugin.Stream(context.Background(), vf)
	assert.Nil(t, err)
	if assert.NotNil(t, rdr2) {
		buf, err := ioutil.ReadAll(rdr2)
		if assert.NoError(t, err) {
			assert.Equal(t, "hello", string(buf))
		}
	}
}

func TestVolumeFileErr(t *testing.T) {
	impl := &mockFileEntry{EntryBase: plugin.NewEntry("parent"), err: errors.New("fail")}
	vf := newFile("mine", plugin.EntryAttributes{}, impl, "my path")

	_, err := vf.Read(context.Background(), nil, 0)
	assert.Equal(t, errors.New("fail"), err)

	rdr2, err := plugin.Stream(context.Background(), vf)
	assert.Nil(t, rdr2)
	assert.Equal(t, errors.New("fail"), err)
}

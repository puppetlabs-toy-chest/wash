package plugin

import (
	"context"
)

// entryContent is the cached result of a Read invocation
type entryContent interface {
	read(context.Context, int64, int64) ([]byte, error)
	size() uint64
}

// entryContentImpl is the default implementation of entryContent,
// meant for Readable entries
type entryContentImpl struct {
	content []byte
}

func newEntryContent(content []byte) *entryContentImpl {
	return &entryContentImpl{
		content: content,
	}
}

func (c *entryContentImpl) read(_ context.Context, size int64, offset int64) (data []byte, err error) {
	data = []byte{}
	contentSize := int64(len(c.content))
	if offset >= contentSize {
		return
	}
	endIx := offset + size
	if contentSize < endIx {
		endIx = contentSize
	}
	data = c.content[offset:endIx]
	return
}

func (c *entryContentImpl) size() uint64 {
	return uint64(len(c.content))
}

// blockReadableEntryContent is the implementation of entryContent that's
// meant for BlockReadable entries. For now, it doesn't cache any content.
type blockReadableEntryContent struct {
	readFunc func(context.Context, int64, int64) ([]byte, error)
	sz       uint64
}

func newBlockReadableEntryContent(readFunc func(context.Context, int64, int64) ([]byte, error)) *blockReadableEntryContent {
	return &blockReadableEntryContent{
		readFunc: readFunc,
	}
}

// Note that we don't need to check the offset/size because if the size attribute
// was set, then plugin.Read already did that validation. If the size attribute
// wasn't set, then it is the responsibility of the plugin's API to raise the error
// for us.
func (c *blockReadableEntryContent) read(ctx context.Context, size int64, offset int64) ([]byte, error) {
	return c.readFunc(ctx, size, offset)
}

func (c *blockReadableEntryContent) size() uint64 {
	return c.sz
}

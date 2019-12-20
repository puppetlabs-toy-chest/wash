package fuse

import (
	"context"
	"testing"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	plugintest "github.com/puppetlabs/wash/plugin/test"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type fileTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *fileTestSuite) SetupTest() {
	plugin.SetTestCache(datastore.NewMemCache())
}

func (suite *fileTestSuite) TearDownTest() {
	plugin.UnsetTestCache()
}

func (suite *fileTestSuite) TestOpen_File() {
	var req fuse.OpenRequest
	var resp fuse.OpenResponse

	m := plugintest.NewMockBase()
	mr := plugintest.NewMockRead()
	mr.Attributes().SetSize(1)
	mw := plugintest.NewMockWrite()
	mw.Attributes().SetSize(1)
	mrw := plugintest.NewMockReadWrite()
	mrw.Attributes().SetSize(1)

	// expectErrs corresponds to expectations on fields in flags.
	flags := []fuse.OpenFlags{fuse.OpenReadOnly, fuse.OpenWriteOnly, fuse.OpenReadWrite}
	for m, expectErrs := range map[plugin.Entry][]bool{
		m:   []bool{true, true, true},
		mr:  []bool{false, true, true},
		mw:  []bool{true, false, true},
		mrw: []bool{false, false, false},
	} {
		f := newFile(nil, m)

		for i, flag := range flags {
			req.Flags = flag
			_, err := f.Open(suite.ctx, &req, &resp)
			if expectErr := expectErrs[i]; expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		}
	}
}

func (suite *fileTestSuite) TestOpen_NonFile() {
	var req fuse.OpenRequest
	var resp fuse.OpenResponse

	mr := plugintest.NewMockRead()
	mr.On("Read", suite.ctx).Return([]byte{'_'}, nil).Once()
	mw := plugintest.NewMockWrite()
	mrw := plugintest.NewMockReadWrite()
	mrw.On("Read", suite.ctx).Return([]byte{'_'}, nil).Once()

	// expectErrs corresponds to expectations on fields in flags.
	flags := []fuse.OpenFlags{fuse.OpenReadOnly, fuse.OpenWriteOnly, fuse.OpenReadWrite}
	for m, expectErrs := range map[plugin.Entry][]bool{
		mr:  []bool{false, true, true},
		mw:  []bool{true, false, true},
		mrw: []bool{false, false, true},
	} {
		f := newFile(nil, m)

		for i, flag := range flags {
			req.Flags = flag
			_, err := f.Open(suite.ctx, &req, &resp)
			if expectErr := expectErrs[i]; expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		}
	}

	mock.AssertExpectationsForObjects(suite.T(), mr, mw, mrw)
}

func (suite *fileTestSuite) TestOpen_NonFileDirect() {
	m := plugintest.NewMockReadWrite()
	m.On("Read", suite.ctx).Return([]byte("hello"), nil).Once()

	f := newFile(nil, m)
	req := fuse.OpenRequest{Flags: fuse.OpenReadOnly}
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Equal(fuse.OpenDirectIO, resp.Flags)
	suite.Equal(f, handle)

	req.Flags = fuse.OpenWriteOnly
	handle, err = f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Equal(fuse.OpenDirectIO, resp.Flags)
	suite.Equal(f, handle)

	var attr fuse.Attr
	err = f.Attr(suite.ctx, &attr)
	suite.NoError(err)
	suite.Equal(uint64(5), attr.Size)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestOpen_FileBuffered() {
	m := plugintest.NewMockReadWrite()
	m.Attributes().SetSize(1)

	f := newFile(nil, m)
	req := fuse.OpenRequest{Flags: fuse.OpenReadOnly}
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Zero(resp.Flags)
	suite.Equal(f, handle)

	req.Flags = fuse.OpenWriteOnly
	handle, err = f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Zero(resp.Flags)
	suite.Equal(f, handle)

	req.Flags = fuse.OpenReadWrite
	handle, err = f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Zero(resp.Flags)
	suite.Equal(f, handle)

	var attr fuse.Attr
	err = f.Attr(suite.ctx, &attr)
	suite.NoError(err)
	suite.Equal(uint64(1), attr.Size)
}

func (suite *fileTestSuite) TestOpenAndRelease() {
	m := plugintest.NewMockReadWrite()
	m.Attributes().SetSize(5)

	f := newFile(nil, m)
	req := fuse.OpenRequest{Flags: fuse.OpenReadWrite}
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Empty(f.writers)
	if suite.Implements((*fs.HandleReleaser)(nil), handle) {
		relReq := fuse.ReleaseRequest{Flags: fuse.OpenReadWrite}
		err = handle.(fs.HandleReleaser).Release(suite.ctx, &relReq)
		suite.NoError(err)
		suite.Empty(f.writers)
	}

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestOpenAndRelease_WriteOnly() {
	m := plugintest.NewMockWrite()

	f := newFile(nil, m)
	req := fuse.OpenRequest{Flags: fuse.OpenWriteOnly}
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Empty(f.writers)
	if suite.Implements((*fs.HandleReleaser)(nil), handle) {
		relReq := fuse.ReleaseRequest{Flags: fuse.OpenReadWrite}
		err = handle.(fs.HandleReleaser).Release(suite.ctx, &relReq)
		suite.NoError(err)
		suite.Empty(f.writers)
	}
}

func (suite *fileTestSuite) TestOpenAndFlush() {
	m := plugintest.NewMockReadWrite()
	m.Attributes().SetSize(5)

	f := newFile(nil, m)
	req := fuse.OpenRequest{Flags: fuse.OpenReadWrite}
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Empty(f.writers)
	if suite.Implements((*fs.HandleFlusher)(nil), handle) {
		err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{})
		suite.NoError(err)
		suite.Empty(f.writers)
		suite.Nil(f.data)
	}
}

func (suite *fileTestSuite) TestOpenAndFlushRelease() {
	m := plugintest.NewMockReadWrite()
	m.Attributes().SetSize(5)

	f := newFile(nil, m)
	req := fuse.OpenRequest{Flags: fuse.OpenReadWrite}
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Empty(f.writers)
	if suite.Implements((*fs.HandleReleaser)(nil), handle) {
		relReq := fuse.ReleaseRequest{Flags: fuse.OpenReadWrite, ReleaseFlags: fuse.ReleaseFlush}
		err = handle.(fs.HandleReleaser).Release(suite.ctx, &relReq)
		suite.NoError(err)
		suite.Empty(f.writers)
		suite.Nil(f.data)
	}
}

func (suite *fileTestSuite) TestAttrWithReaders() {
	m := plugintest.NewMockRead()
	m.Attributes().SetSize(1)

	f := newFile(nil, m)
	req := fuse.OpenRequest{Flags: fuse.OpenReadOnly}
	var resp fuse.OpenResponse
	_, err := f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Empty(f.writers)

	var attr fuse.Attr
	err = f.Attr(suite.ctx, &attr)
	suite.NoError(err)
	suite.Equal(uint64(1), attr.Size)
}

func (suite *fileTestSuite) TestAttrWithWriters() {
	m := plugintest.NewMockWrite()
	m.Attributes().SetSize(5)

	f := newFile(nil, m)
	req := fuse.OpenRequest{Flags: fuse.OpenWriteOnly}
	var resp fuse.OpenResponse
	_, err := f.Open(suite.ctx, &req, &resp)
	suite.NoError(err)
	suite.Empty(f.writers)

	var attr fuse.Attr
	err = f.Attr(suite.ctx, &attr)
	suite.NoError(err)
	suite.Equal(uint64(5), attr.Size)
}

func (suite *fileTestSuite) TestSetAttr_NoHandle() {
	m := plugintest.NewMockReadWrite()
	m.Attributes().SetSize(0)

	f := newFile(nil, m)
	req := fuse.SetattrRequest{Valid: fuse.SetattrSize, Size: 1}
	var resp fuse.SetattrResponse
	err := f.Setattr(suite.ctx, &req, &resp)
	suite.Error(err)
}

func (suite *fileTestSuite) assertFileHandle(handle fs.Handle) bool {
	return suite.Implements((*fs.HandleReader)(nil), handle) &&
		suite.Implements((*fs.HandleWriter)(nil), handle) &&
		suite.Implements((*fs.HandleFlusher)(nil), handle) &&
		suite.Implements((*fs.HandleReleaser)(nil), handle)
}

func (suite *fileTestSuite) TestRead_NonFile() {
	m := plugintest.NewMockRead()
	m.On("Read", suite.ctx).Return([]byte("hello"), nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenReadOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	readReq := fuse.ReadRequest{Offset: 2, Size: 2, Handle: 1}
	var readResp fuse.ReadResponse
	err = handle.(fs.HandleReader).Read(suite.ctx, &readReq, &readResp)
	suite.NoError(err)
	suite.Equal([]byte("ll"), readResp.Data)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	relReq := fuse.ReleaseRequest{ReleaseFlags: fuse.ReleaseFlush, Handle: 1}
	err = handle.(fs.HandleReleaser).Release(suite.ctx, &relReq)
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestRead_File() {
	m := plugintest.NewMockRead()
	m.Attributes().SetSize(5)
	m.On("Read", suite.ctx).Return([]byte("hello"), nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenReadOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	readReq := fuse.ReadRequest{Offset: 2, Size: 2, Handle: 1}
	var readResp fuse.ReadResponse
	err = handle.(fs.HandleReader).Read(suite.ctx, &readReq, &readResp)
	suite.NoError(err)
	suite.Equal([]byte("ll"), readResp.Data)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	relReq := fuse.ReleaseRequest{ReleaseFlags: fuse.ReleaseFlush, Handle: 1}
	err = handle.(fs.HandleReleaser).Release(suite.ctx, &relReq)
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestWrite_NonFile() {
	m := plugintest.NewMockWrite()
	// Called on first Flush only.
	m.On("Write", suite.ctx, []byte("hello")).Return(nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenWriteOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	writeReq := fuse.WriteRequest{Offset: 0, Data: []byte("hello"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.NoError(err)
	suite.Equal(5, writeResp.Size)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	relReq := fuse.ReleaseRequest{ReleaseFlags: fuse.ReleaseFlush, Handle: 1}
	err = handle.(fs.HandleReleaser).Release(suite.ctx, &relReq)
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestWrite_File() {
	m := plugintest.NewMockWrite()
	m.Attributes().SetSize(5)
	// Called on both Flush and Release+Flush only.
	m.On("Write", suite.ctx, []byte("hello")).Return(nil).Twice()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenWriteOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	writeReq := fuse.WriteRequest{Offset: 0, Data: []byte("hello"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.NoError(err)
	suite.Equal(5, writeResp.Size)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	relReq := fuse.ReleaseRequest{ReleaseFlags: fuse.ReleaseFlush, Handle: 1}
	err = handle.(fs.HandleReleaser).Release(suite.ctx, &relReq)
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestTruncateAndWrite_File() {
	m := plugintest.NewMockReadWrite()
	m.Attributes().SetSize(4)
	m.On("Write", suite.ctx, []byte("hello")).Return(nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenWriteOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	setReq := fuse.SetattrRequest{Valid: fuse.SetattrHandle | fuse.SetattrSize, Handle: 1, Size: 0}
	var setResp fuse.SetattrResponse
	err = f.Setattr(suite.ctx, &setReq, &setResp)
	suite.NoError(err)

	writeReq := fuse.WriteRequest{Offset: 0, Data: []byte("hello"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.NoError(err)
	suite.Equal(5, writeResp.Size)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	err = handle.(fs.HandleReleaser).Release(suite.ctx, &fuse.ReleaseRequest{Handle: 1})
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestGrowAndWrite_File() {
	m := plugintest.NewMockReadWrite()
	m.Attributes().SetSize(4)
	m.On("Write", suite.ctx, append([]byte("hello"), make([]byte, 5)...)).Return(nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenWriteOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	setReq := fuse.SetattrRequest{Valid: fuse.SetattrHandle | fuse.SetattrSize, Handle: 1, Size: 10}
	var setResp fuse.SetattrResponse
	err = f.Setattr(suite.ctx, &setReq, &setResp)
	suite.NoError(err)

	writeReq := fuse.WriteRequest{Offset: 0, Data: []byte("hello"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.NoError(err)
	suite.Equal(5, writeResp.Size)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	err = handle.(fs.HandleReleaser).Release(suite.ctx, &fuse.ReleaseRequest{Handle: 1})
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestPartialWrite_NonFile() {
	m := plugintest.NewMockWrite()
	m.On("Write", suite.ctx, []byte{0, 0, '1', '1'}).Return(nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenWriteOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	writeReq := fuse.WriteRequest{Offset: 2, Data: []byte("11"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.NoError(err)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestPartialWrite_File() {
	m := plugintest.NewMockWrite()
	m.Attributes().SetSize(3)

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenWriteOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	writeReq := fuse.WriteRequest{Offset: 2, Data: []byte("11"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.Error(err)
}

func (suite *fileTestSuite) TestReadWrite_FileSmaller() {
	m := plugintest.NewMockBlockReadWrite()
	m.Attributes().SetSize(5)
	m.On("Read", suite.ctx, int64(2), int64(0)).Return([]byte("he"), nil).Once()
	m.On("Write", suite.ctx, []byte("hell")).Return(nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenReadWrite}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	// Shrink the file
	setReq := fuse.SetattrRequest{Valid: fuse.SetattrHandle | fuse.SetattrSize, Handle: 1, Size: 4}
	var setResp fuse.SetattrResponse
	err = f.Setattr(suite.ctx, &setReq, &setResp)
	suite.NoError(err)

	writeReq := fuse.WriteRequest{Offset: 2, Data: []byte("ll"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.NoError(err)
	suite.Equal(2, writeResp.Size)

	// Test this for correct size mid-write. After release, the reported size will be whatever
	// future calls to List return.
	var attr fuse.Attr
	err = f.Attr(suite.ctx, &attr)
	suite.NoError(err)
	suite.Equal(uint64(4), attr.Size)

	err = handle.(fs.HandleReleaser).Release(suite.ctx, &fuse.ReleaseRequest{ReleaseFlags: fuse.ReleaseFlush, Handle: 1})
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestReadWrite_FilePartialWriteOnly() {
	m := plugintest.NewMockReadWrite()
	m.Attributes().SetSize(5)
	m.On("Read", suite.ctx).Return([]byte("hello"), nil).Once()
	m.On("Write", suite.ctx, []byte("he11o")).Return(nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenWriteOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	writeReq := fuse.WriteRequest{Offset: 2, Data: []byte("11"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.NoError(err)
	suite.Equal(2, writeResp.Size)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	// Test this for correct size mid-write. After release, the reported size will be whatever
	// future calls to List return. Before flush, we haven't checked the actual size.
	var attr fuse.Attr
	err = f.Attr(suite.ctx, &attr)
	suite.NoError(err)
	suite.Equal(uint64(5), attr.Size)

	err = handle.(fs.HandleReleaser).Release(suite.ctx, &fuse.ReleaseRequest{Handle: 1})
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestReadWrite_FileLarger() {
	m := plugintest.NewMockBlockReadWrite()
	m.Attributes().SetSize(5)
	m.On("Read", suite.ctx, int64(2), int64(0)).Return([]byte("he"), nil).Once()
	m.On("Write", suite.ctx, []byte("hello there")).Return(nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenReadWrite}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	writeReq := fuse.WriteRequest{Offset: 2, Data: []byte("llo there"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.NoError(err)
	suite.Equal(9, writeResp.Size)

	// Test this for correct size mid-write. After flush, the reported size will be
	// whatever future calls to List return.
	var attr fuse.Attr
	err = f.Attr(suite.ctx, &attr)
	suite.NoError(err)
	suite.Equal(uint64(11), attr.Size)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	err = handle.(fs.HandleReleaser).Release(suite.ctx, &fuse.ReleaseRequest{Handle: 1})
	suite.NoError(err)

	m.AssertExpectations(suite.T())
}

func (suite *fileTestSuite) TestWriteRead_NonFile() {
	m := plugintest.NewMockReadWrite()
	m.On("Write", suite.ctx, []byte("hello there")).Return(nil).Once()
	m.On("Read", suite.ctx).Return([]byte("not what I expected"), nil).Once()

	f := newFile(nil, m)
	var resp fuse.OpenResponse
	handle, err := f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenWriteOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	writeReq := fuse.WriteRequest{Offset: 0, Data: []byte("hello there"), Handle: 1}
	var writeResp fuse.WriteResponse
	err = handle.(fs.HandleWriter).Write(suite.ctx, &writeReq, &writeResp)
	suite.NoError(err)
	suite.Equal(11, writeResp.Size)

	// Test that we get empty size mid-write
	var attr fuse.Attr
	err = f.Attr(suite.ctx, &attr)
	suite.NoError(err)
	suite.Equal(uint64(0), attr.Size)

	err = handle.(fs.HandleFlusher).Flush(suite.ctx, &fuse.FlushRequest{Handle: 1})
	suite.NoError(err)

	err = handle.(fs.HandleReleaser).Release(suite.ctx, &fuse.ReleaseRequest{Handle: 1})
	suite.NoError(err)

	handle, err = f.Open(suite.ctx, &fuse.OpenRequest{Flags: fuse.OpenReadOnly}, &resp)
	if !suite.NoError(err) || !suite.assertFileHandle(handle) {
		suite.FailNow("Unusable handle")
	}

	err = f.Attr(suite.ctx, &attr)
	suite.NoError(err)
	suite.Equal(uint64(19), attr.Size)

	readReq := fuse.ReadRequest{Offset: 0, Size: 19, Handle: 1}
	var readResp fuse.ReadResponse
	err = handle.(fs.HandleReader).Read(suite.ctx, &readReq, &readResp)
	suite.NoError(err)
	suite.Equal([]byte("not what I expected"), readResp.Data)

	m.AssertExpectations(suite.T())
}

func TestFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	suite.Run(t, &fileTestSuite{ctx: ctx})
	cancel()
}

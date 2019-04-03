package fuse

import (
	"context"
	"io"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

// ==== FUSE file Interface ====

type file struct {
	*fuseNode
}

var _ fs.Node = (*file)(nil)
var _ = fs.NodeOpener(&file{})
var _ = fs.NodeGetxattrer(&file{})
var _ = fs.NodeListxattrer(&file{})

func newFile(p plugin.Group, e plugin.Entry) *file {
	return &file{newFuseNode("f", p, e)}
}

// Open a file for reading.
func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	journal.Record(ctx, "FUSE: Open %v", f)

	// Initiate content request and return a channel providing the results.
	log.Infof("FUSE: Opening[jid=%v] %v", journal.GetID(ctx), f)
	if plugin.ReadAction.IsSupportedOn(f.entry) {
		content, err := plugin.CachedOpen(ctx, f.entry.(plugin.Readable))
		if err != nil {
			log.Warnf("FUSE: Error[Open,%v]: %v", f, err)
			journal.Record(ctx, "FUSE: Open %v errored: %v", f, err)
			return nil, err
		}

		log.Infof("FUSE: Opened[jid=%v] %v", journal.GetID(ctx), f)
		journal.Record(ctx, "FUSE: Opened %v", f)
		return &fileHandle{r: content, id: f.String()}, nil
	}
	log.Warnf("FUSE: Error[Open,%v,jid=%v]: cannot open this entry", f, journal.GetID(ctx))
	journal.Record(ctx, "FUSE: Open unsupported on %v", f)
	return nil, fuse.ENOTSUP
}

type fileHandle struct {
	r  io.ReaderAt
	id string
}

var _ fs.Handle = (*fileHandle)(nil)
var _ = fs.HandleReleaser(fileHandle{})
var _ = fs.HandleReader(fileHandle{})

// Release closes the open file.
func (fh fileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	log.Infof("FUSE: Release[jid=%v] %v", journal.GetID(ctx), fh.id)
	journal.Record(ctx, "FUSE: Release %v", fh.id)
	if closer, ok := fh.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Read fills a buffer with the requested amount of data from the file.
func (fh fileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	buf := make([]byte, req.Size)
	n, err := fh.r.ReadAt(buf, req.Offset)
	if err == io.EOF {
		err = nil
	}
	log.Infof("FUSE: Read[jid=%v] %v, %v/%v bytes starting at %v: %v", journal.GetID(ctx), fh.id, n, req.Size, req.Offset, err)
	journal.Record(ctx, "FUSE: Read %v/%v bytes starting at %v from %v: %v", n, req.Size, req.Offset, fh.id, err)
	resp.Data = buf[:n]
	return err
}

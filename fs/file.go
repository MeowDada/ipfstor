package fs

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"syscall"
	"time"

	"bazil.org/fuse"
	fs "bazil.org/fuse/fs"
	"github.com/meowdada/ipfstor/drive"
)

// File implements fs.Node and fs.Handle interface.
type File struct {
	drive.File
	fs *FS
}

// Attr implements fs.Node interface.
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	t, err := time.Parse(time.RFC1123, f.File.Timestamp)
	if err != nil {
		return err
	}

	a.Inode = 2
	a.Mode = 0o644
	a.Size = uint64(f.File.Size)
	a.Atime = t
	a.Ctime = t
	a.Mtime = t
	return nil
}

// ReadAll implements fs.HandleReadAller interface.
func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	return nil, nil
}

// Open implements fs.NodeOpener interface.
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenRequest) (fs.Handle, error) {
	drive := f.fs.core
	rc, err := drive.Get(ctx, f.File.Key)
	if err != nil {
		log.Println(err)
	}
	return &descriptor{
		info:       f.File,
		ReadCloser: rc,
	}, nil
}

// Flush implements fs.HandleFlusher interface.
func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	return nil
}

type descriptor struct {
	info drive.File
	io.ReadCloser
}

func (d *descriptor) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	offset := req.Offset
	rs := d.ReadCloser.(io.ReadSeeker)
	_, err := rs.Seek(offset, io.SeekStart)
	if err != nil {
		log.Println(err)
		return syscall.EINVAL
	}

	lr := io.LimitReader(rs, int64(req.Size))
	b := bytes.NewBuffer(resp.Data)
	_, err = io.Copy(b, lr)
	if err != nil {
		log.Println(err)
		return syscall.EINVAL
	}

	return nil
}

func (d *descriptor) ReadAll(ctx context.Context) ([]byte, error) {
	return ioutil.ReadAll(d.ReadCloser)
}

package fs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/meowdada/ipfstor/drive"
)

type File interface {
	fs.Node
	fs.NodeGetattrer
}

type existingFile struct {
	info drive.File
}

func (ef *existingFile) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Println("existingFile.Attr")
	fillAttr(a, ef.info)
	return nil
}

func (ef *existingFile) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	log.Println("existingFile.Getattr")
	fmt.Println(req.Flags)
	fillAttr(&resp.Attr, ef.info)
	return nil
}

type notExistFile struct {
	core drive.Instance
	key  string
	buf  *bufio.Writer
}

func (ef *notExistFile) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Println("notExistFile.Attr")
	a.Mode = 0666
	return nil
}

func (ef *notExistFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	log.Println("notExistFile.Open")
	ef.buf = bufio.NewWriterSize(&bytes.Buffer{}, 1048576)
	return ef, nil
}

func (ef *notExistFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	log.Println("notExistFile.Write")

	n, err := io.Copy(ef.buf, bytes.NewBuffer(req.Data))
	if err != nil {
		return err
	}

	resp.Size = int(n)
	return nil
}

func (ef *notExistFile) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	log.Println("notExistFile.Release")
	if ef.buf != nil {
		return nil
	}

	if err := ef.buf.Flush(); err != nil {
		return err
	}

	return nil
}

func fillAttr(a *fuse.Attr, info drive.File) {
	t, _ := time.Parse(info.Timestamp, time.RFC1123)
	a.Atime = t
	a.Mtime = t
	a.Ctime = t
	a.Size = uint64(info.Size)
	a.Mode = 0644
}

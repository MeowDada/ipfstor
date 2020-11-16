package drive

import (
	"fmt"
	"io"
	"os"
	"time"

	files "github.com/ipfs/go-ipfs-files"
)

func newFile(key string, r io.Reader) files.Node {
	si := &streamInfo{
		name: key,
		size: 0,
	}
	s := newStream(r, si)
	return files.NewReaderStatFile(s, si)
}

type stream struct {
	r    io.Reader
	info *streamInfo
	off  int64
}

func newStream(r io.Reader, info *streamInfo) *stream {
	return &stream{
		r:    r,
		info: info,
		off:  0,
	}
}

func (s *stream) Read(b []byte) (int, error) {
	n, e := s.r.Read(b)
	s.off += int64(n)
	s.info.size = s.off
	return n, e
}

func (s *stream) Seek(off int64, whence int) (int64, error) {
	if sk, ok := s.r.(io.Seeker); ok {
		ns, err := sk.Seek(off, whence)
		if err != nil {
			return 0, err
		}
		s.off = ns
		s.info.size = s.off
		return ns, err
	}
	return 0, fmt.Errorf("underlying intstnace does not unsupport io.Seeker interface")
}

func (s *stream) Close() error {
	if c, ok := s.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// streamInfo implements os.FileInfo interface.
type streamInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	mtime time.Time
	src   *stream
}

func (si *streamInfo) Name() string       { return si.name }
func (si *streamInfo) Size() int64        { return si.size }
func (si *streamInfo) Mode() os.FileMode  { return si.mode }
func (si *streamInfo) ModTime() time.Time { return si.mtime }
func (si *streamInfo) IsDir() bool        { return false }
func (si *streamInfo) Sys() interface{}   { return si.src }

package fs

import (
	"context"
	"os"
	"syscall"

	"bazil.org/fuse"
	fs "bazil.org/fuse/fs"
)

// Dir denotes a directory in this filesystem.
type Dir struct {
	fs *FS
}

// Attr implements fs.Node interface.
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0o555
	return nil
}

// Lookup implements fs.NodeStringLookuper interface.
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	drive := d.fs.core
	f, err := drive.Stat(ctx, name)
	if err != nil {
		return nil, syscall.ENOENT
	}
	return &File{File: f, fs: d.fs}, nil
}

// ReadDirAll implements fs.HandleReadDirAller interface.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	lr, err := d.fs.core.List(ctx, "")
	if err != nil {
		return nil, err
	}

	var dirs []fuse.Dirent
	for _, f := range lr.Files() {
		dirs = append(dirs, fuse.Dirent{
			Inode: 0,
			Type:  fuse.DT_File,
			Name:  f.Key,
		})
	}

	return dirs, nil
}

// Getxattr implements fs.NodeGetxattrer interface.
func (d *Dir) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	return fuse.ErrNoXattr
}

package fs

import (
	"context"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/meowdada/ipfstor/drive"
)

// Dir denotes a directory in this filesystem.
type Dir struct {
	core drive.Instance
}

// Attr implements fs.Node interface.
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Println("dir.Attr")
	a.Inode = 1
	a.Mode = os.ModeDir | 0755
	return nil
}

// Lookup implements fs.NodeLookuper interface.
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	log.Println("dir.Lookup")
	info, err := d.core.Stat(ctx, name)
	if err != nil {
		return &notExistFile{
			core: d.core,
			key:  name,
		}, nil
	}
	return &existingFile{info: info}, nil
}

// ReadDirAll implements fs.HandleReadDirAller interface.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	log.Println("dir.ReadDirAll")
	lr, err := d.core.List(ctx, "")
	if err != nil {
		log.Println(err)
		return nil, nil
	}

	var dirs []fuse.Dirent
	for _, r := range lr.Files() {
		dirs = append(dirs, fuse.Dirent{
			Type: fuse.DT_File,
			Name: r.Key,
		})
	}
	return dirs, nil
}

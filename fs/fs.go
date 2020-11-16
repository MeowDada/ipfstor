package fs

import (
	"context"

	"bazil.org/fuse"
	fs "bazil.org/fuse/fs"
	"github.com/meowdada/ipfstor/drive"
)

// Mount mounts the filesystem to specific path.
func Mount(mountpoint string, drive drive.Instance) error {
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("QNAP-ipfs"),
		fuse.Subtype("qpfs"),
	)
	if err != nil {
		return err
	}
	defer c.Close()

	return fs.Serve(c, &FS{
		core: drive,
	})
}

// FS denotes a file system instance backed by a existing drive.
type FS struct {
	core drive.Instance
}

// Root implements fs.FS interface.
func (fs *FS) Root() (fs.Node, error) {
	return &Dir{}, nil
}

// Statfs implements fs.FSStatfser interface.
func (fs *FS) Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error {
	return nil
}

// Destroy implements fs.FSDestroyer interface.
func (fs *FS) Destroy() {}

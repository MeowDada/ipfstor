package fs

import (
	"context"
	"fmt"
	"syscall"

	"bazil.org/fuse"
	fs "bazil.org/fuse/fs"
	"github.com/meowdada/ipfstor/drive"
)

// Mount mounts the filesystem to specific path.
func Mount(mountpoint string, drive drive.Instance) (unmountFn func(), err error) {
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("QNAP-ipfs"),
		fuse.Subtype("qpfs"),
	)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	if err := fs.Serve(c, &FS{
		mountpoint: mountpoint,
		core:       drive,
	}); err != nil {
		return nil, err
	}

	return func() {
		if err := syscall.Unmount(mountpoint, 0); err != nil {
			fmt.Println(err)
		}
	}, nil
}

// FS denotes a file system instance backed by a existing drive.
type FS struct {
	mountpoint string
	core       drive.Instance
}

// Root implements fs.FS interface.
func (fs *FS) Root() (fs.Node, error) {
	return &Dir{
		fs: fs,
	}, nil
}

// Statfs implements fs.FSStatfser interface.
func (fs *FS) Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error {
	return nil
}

// Destroy implements fs.FSDestroyer interface.
func (fs *FS) Destroy() {
	fmt.Println("FS instance has been removed successfully")
}

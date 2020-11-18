package fs

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"bazil.org/fuse"
	fs "bazil.org/fuse/fs"
	"github.com/meowdada/ipfstor/drive"
)

// Mount mounts the filesystem to specific path.
func Mount(mountpoint string, drive drive.Instance) (func(), error) {
	conn, err := fuse.Mount(
		mountpoint,
		fuse.FSName("QNAP-ipfs"),
		fuse.Subtype("qpfs"),
		fuse.AllowOther(),
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)

	var once sync.Once
	unmount := func() {
		once.Do(func() {
			if err := fuse.Unmount(mountpoint); err != nil {
				fmt.Println(err)
			}
		})
	}

	// Monitor routine to handle interrupt signal.
	go func() {
		select {
		case <-ctx.Done():
			return
		case sig := <-ch:
			fmt.Println("reciving signal:", sig)
			unmount()
		}
	}()

	if err := fs.Serve(conn, &FS{
		mountpoint: mountpoint,
		core:       drive,
	}); err != nil {
		return nil, err
	}

	return unmount, nil
}

// FS denotes a file system instance backed by a existing drive.
type FS struct {
	mountpoint string
	core       drive.Instance
}

// Root implements fs.FS interface.
func (fs *FS) Root() (fs.Node, error) {
	return &Dir{core: fs.core}, nil
}

package fs

import (
	"context"
	"testing"

	"github.com/meowdada/ipfstor/drive"
	"github.com/meowdada/ipfstor/options"
	"github.com/stretchr/testify/require"
)

func TestMount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d, err := drive.DirectOpen("www", options.OpenDrive().SetCreate(true))
	require.NoError(t, err)
	defer d.Close(ctx)

	mountpoint := "/home/jack/Desktop/fuse"

	unmount, err := Mount(mountpoint, d)
	require.NoError(t, err)
	defer unmount()
}

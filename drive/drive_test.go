package drive

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	ipfsCore "github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	mock "github.com/ipfs/go-ipfs/core/mock"
	iface "github.com/ipfs/interface-go-ipfs-core"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/meowdada/ipfstor/options"
	"github.com/stretchr/testify/require"
)

func mockIPFSNode(ctx context.Context, t *testing.T, m mocknet.Mocknet) (*ipfsCore.IpfsNode, func()) {
	t.Helper()

	core, err := ipfsCore.NewNode(ctx, &ipfsCore.BuildCfg{
		Online: true,
		Host:   mock.MockHostOption(m),
		ExtraOpts: map[string]bool{
			"pubsub": true,
		},
	})
	require.NoError(t, err)

	cleanup := func() { core.Close() }
	return core, cleanup
}

func mockAPI(t *testing.T, core *ipfsCore.IpfsNode) iface.CoreAPI {
	t.Helper()

	api, err := coreapi.NewCoreAPI(core)
	require.NoError(t, err)
	return api
}

func mockNet(ctx context.Context) mocknet.Mocknet {
	return mocknet.New(ctx)
}

func mockTempDir(t *testing.T, name string) (string, func()) {
	t.Helper()

	path, err := ioutil.TempDir("", name)
	require.NoError(t, err)

	cleanup := func() { os.RemoveAll(path) }
	return path, cleanup
}

func mockDrive(t *testing.T, resolve string) (Instance, func()) {
	ctx := context.Background()
	_, dbPathClean := mockTempDir(t, "db")
	net := mockNet(ctx)
	node, nodeClean := mockIPFSNode(ctx, t, net)
	ipfs := mockAPI(t, node)

	opts := options.OpenDrive().SetCreate(true)

	d, err := Open(ctx, ipfs, resolve, opts)
	require.NoError(t, err)

	return d, func() {
		d.Close(ctx)
		nodeClean()
		dbPathClean()
	}
}

func mockFile(t *testing.T, key string, content []byte) func() {
	f, err := os.Create(key)
	require.NoError(t, err)

	_, err = io.Copy(f, bytes.NewBuffer(content))
	require.NoError(t, err)

	return func() {
		f.Close()
		os.Remove(key)
	}
}

func TestOpenDrive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		ipfs iface.CoreAPI
	)

	setup := func(t *testing.T) func() {
		_, dbPathClean := mockTempDir(t, "db")
		net := mockNet(ctx)
		node, nodeClean := mockIPFSNode(ctx, t, net)
		ipfs = mockAPI(t, node)

		cleanup := func() {
			nodeClean()
			dbPathClean()
		}
		return cleanup
	}

	t.Run("Open an unexisting drive", func(t *testing.T) {
		defer setup(t)()
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		_, err := Open(ctx, ipfs, "gfd")
		require.NotNil(t, err)
	})

	t.Run("Create an existing drive", func(t *testing.T) {
		defer setup(t)()
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		opts := options.OpenDrive().SetCreate(true)

		d, err := Open(ctx, ipfs, "gfd", opts)
		require.NoError(t, err)
		defer d.Close(ctx)
	})

	t.Run("Open a drive with empty key", func(t *testing.T) {
		defer setup(t)()
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		opts := options.OpenDrive().SetCreate(true)

		_, err := Open(ctx, ipfs, "", opts)
		require.NotNil(t, err)
	})

	t.Run("Open a drive with nil ipfs API", func(t *testing.T) {
		defer setup(t)()
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		opts := options.OpenDrive().SetCreate(true)

		_, err := Open(ctx, nil, "gfd", opts)
		require.NotNil(t, err)
	})

	t.Run("Open drive directly", func(t *testing.T) {
		opts := options.OpenDrive().SetCreate(true)
		_, err := DirectOpen("gfd", opts)
		require.NoError(t, err)
	})
}

const (
	mockDriveName = "tsmc"
)

func TestDriveName(t *testing.T) {
	d, cleanup := mockDrive(t, mockDriveName)
	defer cleanup()
	require.Equal(t, d.Name(), mockDriveName)
}

func TestDriveAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, dbPathClean := mockTempDir(t, "db")
	net := mockNet(ctx)
	node, nodeClean := mockIPFSNode(ctx, t, net)
	ipfs := mockAPI(t, node)

	opts := options.OpenDrive().SetCreate(true)

	db, err := newOrbitDB(ctx, ipfs, opts)
	require.NoError(t, err)

	kv, err := openKeyValueStore(ctx, db, "tsmc", opts)
	require.NoError(t, err)

	d := Raw(db, kv)
	require.Equal(t, d.Address(), kv.Address().String())

	defer func() {
		db.Close()
		nodeClean()
		dbPathClean()
	}()
}

func TestDriveIdentity(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, dbPathClean := mockTempDir(t, "db")
	net := mockNet(ctx)
	node, nodeClean := mockIPFSNode(ctx, t, net)
	ipfs := mockAPI(t, node)

	opts := options.OpenDrive().SetCreate(true)

	db, err := newOrbitDB(ctx, ipfs, opts)
	require.NoError(t, err)

	kv, err := openKeyValueStore(ctx, db, "tsmc", opts)
	require.NoError(t, err)

	d := Raw(db, kv)
	require.Equal(t, d.Identity(), db.Identity().ID)

	defer func() {
		db.Close()
		nodeClean()
		dbPathClean()
	}()
}

func TestDriveAddFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		ipfs iface.CoreAPI
	)

	setup := func(t *testing.T) func() {
		_, dbPathClean := mockTempDir(t, "db")
		net := mockNet(ctx)
		node, nodeClean := mockIPFSNode(ctx, t, net)
		ipfs = mockAPI(t, node)

		cleanup := func() {
			nodeClean()
			dbPathClean()
		}
		return cleanup
	}

	t.Run("Add file normally", func(t *testing.T) {
		defer setup(t)()

		opts := options.OpenDrive().SetCreate(true)

		d, err := Open(ctx, ipfs, mockDriveName, opts)
		require.NoError(t, err)
		defer d.Close(ctx)

		key := "testFile"
		content := []byte("abc")
		close := mockFile(t, key, content)
		defer close()

		_, err = d.AddFile(ctx, key, key)
		require.NoError(t, err)
	})

	t.Run("Add inexisting file", func(t *testing.T) {
		defer setup(t)()

		opts := options.OpenDrive().SetCreate(true)

		d, err := Open(ctx, ipfs, mockDriveName, opts)
		require.NoError(t, err)
		defer d.Close(ctx)

		key := "asgasg"
		path := "sagasgasmg;l"
		_, err = d.AddFile(ctx, key, path)
		require.NotNil(t, err)
	})

	t.Run("Add file with empty key", func(t *testing.T) {
		defer setup(t)()

		opts := options.OpenDrive().SetCreate(true)

		d, err := Open(ctx, ipfs, mockDriveName, opts)
		require.NoError(t, err)
		defer d.Close(ctx)

		key := "testFile"
		content := []byte("abc")
		close := mockFile(t, key, content)
		defer close()

		_, err = d.AddFile(ctx, "", key)
		require.NotNil(t, err)
	})
}

func TestDriveAdd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		ipfs iface.CoreAPI
	)

	setup := func(t *testing.T) func() {
		_, dbPathClean := mockTempDir(t, "db")
		net := mockNet(ctx)
		node, nodeClean := mockIPFSNode(ctx, t, net)
		ipfs = mockAPI(t, node)

		cleanup := func() {
			nodeClean()
			dbPathClean()
		}
		return cleanup
	}

	t.Run("Add file normally", func(t *testing.T) {
		defer setup(t)()

		opts := options.OpenDrive().SetCreate(true)

		d, err := Open(ctx, ipfs, mockDriveName, opts)
		require.NoError(t, err)
		defer d.Close(ctx)

		key := "abc"
		b := bytes.NewBufferString("123")
		size := int64(b.Len())
		f, err := d.Add(ctx, key, b)
		require.NoError(t, err)

		require.Equal(t, f.Size, size)
	})

	t.Run("Add file with empty key", func(t *testing.T) {
		defer setup(t)()

		opts := options.OpenDrive().SetCreate(true)

		d, err := Open(ctx, ipfs, mockDriveName, opts)
		require.NoError(t, err)
		defer d.Close(ctx)

		_, err = d.Add(ctx, "", bytes.NewBuffer(nil))
		require.NotNil(t, err)
	})

	t.Run("Add file with nil reader", func(t *testing.T) {
		defer setup(t)()

		opts := options.OpenDrive().SetCreate(true)

		d, err := Open(ctx, ipfs, mockDriveName, opts)
		require.NoError(t, err)
		defer d.Close(ctx)

		_, err = d.Add(ctx, "abc", nil)
		require.NotNil(t, err)
	})

}

func TestDriveGet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		ipfs iface.CoreAPI
	)

	setup := func(t *testing.T) func() {
		_, dbPathClean := mockTempDir(t, "db")
		net := mockNet(ctx)
		node, nodeClean := mockIPFSNode(ctx, t, net)
		ipfs = mockAPI(t, node)

		cleanup := func() {
			nodeClean()
			dbPathClean()
		}
		return cleanup
	}

	t.Run("Get file normally", func(t *testing.T) {
		defer setup(t)()

		opts := options.OpenDrive().SetCreate(true)

		d, err := Open(ctx, ipfs, mockDriveName, opts)
		require.NoError(t, err)
		defer d.Close(ctx)

		key := "abc"
		content := []byte("123")
		r := bytes.NewBuffer(content)

		_, err = d.Add(ctx, key, r)
		require.NoError(t, err)

		rc, err := d.Get(ctx, key)
		require.NoError(t, err)

		get, err := ioutil.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, content, get)
	})
}

func TestDriveStat(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		ipfs iface.CoreAPI
	)

	setup := func(t *testing.T) func() {
		_, dbPathClean := mockTempDir(t, "db")
		net := mockNet(ctx)
		node, nodeClean := mockIPFSNode(ctx, t, net)
		ipfs = mockAPI(t, node)

		cleanup := func() {
			nodeClean()
			dbPathClean()
		}
		return cleanup
	}

	t.Run("Stat unexisting file", func(t *testing.T) {
		defer setup(t)()

		opts := options.OpenDrive().SetCreate(true)

		d, err := Open(ctx, ipfs, mockDriveName, opts)
		require.NoError(t, err)
		defer d.Close(ctx)

		key := "tsmc"
		_, err = d.Stat(ctx, key)
		require.NotNil(t, err)
	})
}

func TestDriveList(t *testing.T) {

}

func TestDriveRemove(t *testing.T) {

}

func TestDriveGrant(t *testing.T) {

}

func TestDriveRevoke(t *testing.T) {

}

func TestDriveClose(t *testing.T) {

}

/*
func TestDriveReplicate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		ipfs1, ipfs2     iface.CoreAPI
		dbPath1, dbPath2 string
	)

	setup := func(t *testing.T) func() {
		var dbPath1Clean, dbPath2Clean func()
		dbPath1, dbPath1Clean = mockTempDir(t, "db1")
		dbPath2, dbPath2Clean = mockTempDir(t, "db2")

		mockNet := mockNet(ctx)

		node1, node1Clean := mockIPFSNode(ctx, t, mockNet)
		node2, node2Clean := mockIPFSNode(ctx, t, mockNet)

		ipfs1 = mockAPI(t, node1)
		ipfs2 = mockAPI(t, node2)

		zap.L().Named("orbitdb.tests").Debug(fmt.Sprintf("node1 is %s", node1.Identity.String()))
		zap.L().Named("orbitdb.tests").Debug(fmt.Sprintf("node2 is %s", node2.Identity.String()))

		_, err := mockNet.LinkPeers(node1.Identity, node2.Identity)
		require.NoError(t, err)

		peerInfo1 := peer.AddrInfo{ID: node1.Identity, Addrs: node1.PeerHost.Addrs()}
		err = ipfs2.Swarm().Connect(ctx, peerInfo1)
		require.NoError(t, err)

		peerInfo2 := peer.AddrInfo{ID: node2.Identity, Addrs: node2.PeerHost.Addrs()}
		err = ipfs1.Swarm().Connect(ctx, peerInfo2)
		require.NoError(t, err)

		cleanup := func() {
			node1Clean()
			node2Clean()
			dbPath1Clean()
			dbPath2Clean()
		}
		return cleanup
	}

	t.Run("automatic replication", func(t *testing.T) {
		defer setup(t)()

		drive1, err := Open(ctx, ipfs1, "replication-test", options.OpenDrive().SetDirectory(dbPath1).SetCreate(true))
		require.NoError(t, err)

		drive2, err := Open(ctx, ipfs2, drive1.Address(), options.OpenDrive().SetDirectory(dbPath2).SetCreate(true))
		require.NoError(t, err)

		_, err = drive1.Add(ctx, "123", "/home/jack/Desktop/data/file1.tmp")
		require.NoError(t, err)

		time.Sleep(5 * time.Second)

		var r ListResult
		r, err = drive2.List(ctx, "")
		require.NoError(t, err)

		fmt.Println(r.Files())
	})
}
*/

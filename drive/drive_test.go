package drive

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	ipfsCore "github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	mock "github.com/ipfs/go-ipfs/core/mock"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/meowdada/ipfstor/options"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

func TestListResultWriteTo(t *testing.T) {
	lr := ListResult{
		files: []File{
			{"hello-world", cid.Cid{}, 1024, time.Now(), "abc"},
			{"abs", cid.Cid{}, 1048579, time.Now(), "qde"},
			{"hello-world123", cid.Cid{}, 54121561, time.Now(), "def"},
			{"mamaytata", cid.Cid{}, 123, time.Now(), "asbhash"},
		},
	}

	fmt.Println(string(lr.Bytes(ListMaskKey | ListMaskSize)))
}

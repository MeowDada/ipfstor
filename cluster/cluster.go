package cluster

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	ds "github.com/ipfs/go-datastore"
	ipfscluster "github.com/ipfs/ipfs-cluster"
	"github.com/ipfs/ipfs-cluster/allocator/descendalloc"
	"github.com/ipfs/ipfs-cluster/api/ipfsproxy"
	"github.com/ipfs/ipfs-cluster/api/rest"
	clnt "github.com/ipfs/ipfs-cluster/api/rest/client"
	"github.com/ipfs/ipfs-cluster/cmdutils"
	"github.com/ipfs/ipfs-cluster/config"
	"github.com/ipfs/ipfs-cluster/consensus/crdt"
	"github.com/ipfs/ipfs-cluster/informer/disk"
	"github.com/ipfs/ipfs-cluster/ipfsconn/ipfshttp"
	"github.com/ipfs/ipfs-cluster/monitor/pubsubmon"
	"github.com/ipfs/ipfs-cluster/observations"
	"github.com/ipfs/ipfs-cluster/pintracker/stateless"
	"github.com/ipfs/ipfs-cluster/pstoremgr"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/meowdada/ipfstor/options"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	clusterConsensus = "crdt"

	ipfsClusterRootDirName = ".ipfs-cluster"

	defaultConfigName   = "service.json"
	defaultIdentityName = "identity.json"
)

var (
	// ErrInvalConfigPath denotes an invalid config path error.
	ErrInvalConfigPath = errors.New("invalid config path")

	// ErrInvalIdentityPath denotes an invalid identity path error.
	ErrInvalIdentityPath = errors.New("invalid ientity path")
)

// DefaultConfigPath returns the default path to the cluster config file.
func DefaultConfigPath() string {
	dir, _ := homedir.Dir()
	return filepath.Join(dir, ipfsClusterRootDirName, defaultConfigName)
}

// DefaultIdentityPath returns the default path to the identity file.
func DefaultIdentityPath() string {
	dir, _ := homedir.Dir()
	return filepath.Join(dir, ipfsClusterRootDirName, defaultIdentityName)
}

// Cluster denotes an instance of ipfs cluster.
type Cluster struct {
	*ipfscluster.Cluster
	host  host.Host
	dht   *dual.DHT
	store ds.Datastore
	exit  chan struct{}
}

// Bootstrap bootstraps given peers to the cluster.
func (cls Cluster) Bootstrap(ctx context.Context, peerAddrs []string) error {
	var multiAddrs []ma.Multiaddr
	for i := range peerAddrs {
		multiAddr, err := ma.NewMultiaddr(peerAddrs[i])
		if err != nil {
			return err
		}
		multiAddrs = append(multiAddrs, multiAddr)
	}

	for i := range multiAddrs {
		if err := cls.Cluster.Join(ctx, multiAddrs[i]); err != nil {
			return err
		}
	}

	return nil
}

// Start the cluster.
func (cls Cluster) Start() {
	go cls.start()
}

// Stop stops the cluster.
func (cls Cluster) Stop(ctx context.Context) error {
	if err := cls.Cluster.Shutdown(ctx); err != nil {
		return err
	}
	fmt.Println("cluster has been shutdown successfully")
	return nil
}

func (cls Cluster) start() {
	sigs := make(chan os.Signal, 20)
	signal.Notify(
		sigs,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)

	for {
		select {
		case <-sigs:
			err := cls.Cluster.Shutdown(context.Background())
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		case <-cls.Cluster.Done():
			err := multierr.Combine(
				cls.dht.Close(),
				cls.host.Close(),
				cls.store.Close(),
			)
			if err != nil {
				fmt.Println(err)
			}
			return
		}
	}
}

// NewClient creates an instance of cluster client.
func NewClient(ctx context.Context, addr string) (clnt.Client, error) {
	maAddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}
	return clnt.NewDefaultClient(&clnt.Config{
		APIAddr: maAddr,
	})
}

// NewConfig creates a cluster config and a identity file with given options.
func NewConfig(ctx context.Context, opts ...*options.ClusterOption) error {
	opt := options.MergeClusterOptions(opts...)

	if opt.ConfigPath == nil {
		return ErrInvalConfigPath
	}

	if opt.IdentityPath == nil {
		return ErrInvalIdentityPath
	}

	configPath := *opt.ConfigPath
	identityPath := *opt.IdentityPath
	consensus := clusterConsensus

	cfgHelper := cmdutils.NewConfigHelper(configPath, identityPath, consensus)
	defer cfgHelper.Manager().Shutdown()

	var err error
	cfgs := cfgHelper.Configs()

	identityExists := false
	if _, err := os.Stat(identityPath); !os.IsNotExist(err) {
		identityExists = true
	}

	/*
		configPathExists := false
		if _, err := os.Stat(configPath); !os.IsNotExist(err) {
			configPathExists = true
		}
	*/

	// Sets source url.
	if opt.SrcURL != nil {
		cfgHelper.Manager().Source = *opt.SrcURL
	}

	// Generate default settings.
	if err := cfgHelper.Manager().Default(); err != nil {
		return err
	}

	// Using random ports if the option is enabled.
	if opt.RandomPorts != nil && *opt.RandomPorts {
		cfgs.Cluster.ListenAddr, err = cmdutils.RandomizePorts(cfgs.Cluster.ListenAddr)
		if err != nil {
			return err
		}
		cfgs.Restapi.HTTPListenAddr, err = cmdutils.RandomizePorts(cfgs.Restapi.HTTPListenAddr)
		if err != nil {
			return err
		}
		cfgs.Ipfsproxy.ListenAddr, err = cmdutils.RandomizePorts(cfgs.Ipfsproxy.ListenAddr)
		if err != nil {
			return err
		}
	}

	if err := cfgHelper.Manager().ApplyEnvVars(); err != nil {
		return err
	}

	if opt.Secret != nil {
		decodedSecret, err := ipfscluster.DecodeClusterSecret(*opt.Secret)
		if err != nil {
			return err
		}
		cfgHelper.Configs().Cluster.Secret = decodedSecret
	}

	var multiAddrs []ma.Multiaddr
	for i := range opt.PeerAddrs {
		multiAddr, err := ma.NewMultiaddr(opt.PeerAddrs[i])
		if err != nil {
			return err
		}
		multiAddrs = append(multiAddrs, multiAddr)
	}

	if len(opt.PeerAddrs) > 0 {
		peers := ipfscluster.PeersFromMultiaddrs(multiAddrs)
		cfgHelper.Configs().Crdt.TrustAll = false
		cfgHelper.Configs().Crdt.TrustedPeers = peers
		cfgHelper.Configs().Raft.InitPeerset = peers
	}

	// Saves the config to the given path.
	if err := cfgHelper.SaveConfigToDisk(); err != nil {
		return err
	}

	if !identityExists {
		ident := cfgHelper.Identity()
		if err := ident.Default(); err != nil {
			return err
		}
		if err := ident.ApplyEnvVars(); err != nil {
			return err
		}
		if err := cfgHelper.SaveIdentityToDisk(); err != nil {
			return err
		}
	}

	// Initialize peerstore file even if it is empty.
	peerstorePath := cfgHelper.Configs().Cluster.GetPeerstorePath()
	peerManager := pstoremgr.New(ctx, nil, peerstorePath)
	addrInfos, err := peer.AddrInfosFromP2pAddrs(multiAddrs...)
	if err != nil {
		return err
	}

	return peerManager.SavePeerstore(addrInfos)
}

// New creates a cluster instance.
func New(ctx context.Context, configPath, identityPath string, peerAddrs []string) (Cluster, error) {
	var bootstraps []ma.Multiaddr
	for i := range peerAddrs {
		multiAddr, err := ma.NewMultiaddr(peerAddrs[i])
		if err != nil {
			return Cluster{}, err
		}
		bootstraps = append(bootstraps, multiAddr)
	}

	cfgHelper, err := cmdutils.NewLoadedConfigHelper(configPath, identityPath)
	if err != nil {
		return Cluster{}, err
	}
	defer cfgHelper.Manager().Shutdown()

	cfgs := cfgHelper.Configs()

	crdtCfg := cfgs.Crdt
	crdtCfg.TrustedPeers = append(crdtCfg.TrustedPeers, ipfscluster.PeersFromMultiaddrs(bootstraps)...)

	store, err := setupDatastore(cfgHelper)
	if err != nil {
		return Cluster{}, err
	}

	host, pubsub, dht, err := ipfscluster.NewClusterHost(ctx, cfgHelper.Identity(), cfgs.Cluster, store)
	if err != nil {
		return Cluster{}, err
	}

	cluster, err := createCluster(ctx, cfgHelper, host, pubsub, dht, store)
	if err != nil {
		return Cluster{}, err
	}

	return Cluster{
		Cluster: cluster,
		host:    host,
		dht:     dht,
		store:   store,
	}, nil

	/*
		go bootstrap(ctx, cluster, bootstraps)

		return cmdutils.HandleSignals(ctx, cancel, cluster, host, dht, store)
	*/
}

func setupDatastore(cfgHelper *cmdutils.ConfigHelper) (ds.Datastore, error) {
	stmgr, err := cmdutils.NewStateManager(clusterConsensus, cfgHelper.Identity(), cfgHelper.Configs())
	if err != nil {
		return nil, err
	}
	return stmgr.GetStore()
}

func setupConsensus(
	cfgHelper *cmdutils.ConfigHelper,
	h host.Host,
	dht *dual.DHT,
	pubsub *pubsub.PubSub,
	store ds.Datastore,
) (ipfscluster.Consensus, error) {
	cfgs := cfgHelper.Configs()
	return crdt.New(h, dht, pubsub, cfgs.Crdt, store)
}

func bootstrap(ctx context.Context, cluster *ipfscluster.Cluster, bootstraps []ma.Multiaddr) error {
	for _, bstrap := range bootstraps {
		err := cluster.Join(ctx, bstrap)
		if err != nil {
			return err
		}
	}
	return nil
}

func createCluster(ctx context.Context, cfgHelper *cmdutils.ConfigHelper, host host.Host, pubsub *pubsub.PubSub, dht *dual.DHT, store ds.Datastore) (*ipfscluster.Cluster, error) {
	cfgs := cfgHelper.Configs()
	cfgMgr := cfgHelper.Manager()

	var apis []ipfscluster.API
	var err error
	if cfgMgr.IsLoadedFromJSON(config.API, cfgs.Restapi.ConfigKey()) {
		var api *rest.API
		api, err = rest.NewAPI(ctx, cfgs.Restapi)
		if err != nil {
			return nil, err
		}
		apis = append(apis, api)
	}

	if cfgMgr.IsLoadedFromJSON(config.API, cfgs.Ipfsproxy.ConfigKey()) {
		proxy, err := ipfsproxy.New(cfgs.Ipfsproxy)
		if err != nil {
			return nil, err
		}
		apis = append(apis, proxy)
	}

	connector, err := ipfshttp.NewConnector(cfgs.Ipfshttp)
	if err != nil {
		return nil, err
	}

	informer, err := disk.NewInformer(cfgs.Diskinf)
	if err != nil {
		return nil, err
	}

	alloc := descendalloc.NewAllocator()

	ipfscluster.ReadyTimeout = cfgs.Raft.WaitForLeaderTimeout + 5*time.Second

	if err := observations.SetupMetrics(cfgs.Metrics); err != nil {
		return nil, err
	}

	tracer, err := observations.SetupTracing(cfgs.Tracing)
	if err != nil {
		return nil, err
	}

	cons, err := setupConsensus(cfgHelper, host, dht, pubsub, store)
	if err != nil {
		return nil, err
	}

	var peersF func(context.Context) ([]peer.ID, error)
	tracker := stateless.New(cfgs.Statelesstracker, host.ID(), cfgs.Cluster.Peername, cons.State)

	mon, err := pubsubmon.New(ctx, cfgs.Pubsubmon, pubsub, peersF)
	if err != nil {
		store.Close()
		return nil, err
	}

	return ipfscluster.NewCluster(
		ctx,
		host,
		dht,
		cfgs.Cluster,
		store,
		cons,
		apis,
		connector,
		tracker,
		mon,
		alloc,
		[]ipfscluster.Informer{informer},
		tracer,
	)
}

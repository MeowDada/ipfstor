# Introudction
This is a package that helps developer to create a cloud-drive style storage with IPFS.

# Usage
The main idea to use this package is to create a `Driver` instance. Which provides useful APIs to perform cloud-drive style operations.

To create a Driver instance, we need to import this package first.
```go
import "github.com/meowdada/ipfstor"
```

In addition, you also need to obtain an IPFS api instance. There are mainly two ways to do this. Check belows subchapter for detail information:

## Created by an existing IPFS daemon (Recommended)
Say you have a running IPFS daemon on your machine. We can create an IPFS API instance bakced by HTTP client.

Import the ipfs http client package.
```go
import ipfsClient "github.com/ipfs/go-ipfs-http-client"
```

Then create an driver instance backed by this api instance. 
```go
// Using http client as backend of IPFS API.
api, err := ipfsClient.NewLocalApi()
if err != nil {
    panic(err)
}

ctx := context.Background()
driverName := "kvstore"

// Create a driver instance with given name. If the driver
// name exist, it will open it instead of creating a new one.
driver, err := ipfstor.NewDriver(ctx, api, driverName)
if err != nil {
    panic(err)
}
```

## Created by embedded IPFS node.
If you don't have a running IPFS node. You can still create an IPFS API instance by embedded go codes.

First, import following packages.
```go
import (
    "github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
)
```

Create or open a repo (default path is $HOME/.ipfs) and initializes the plugins. Then, create an IPFS node backed by the repo. And finally create an IPFS API instance denoted by this node.

```go
repoPath := filepath.Join(os.Getenv("HOME"), ".ipfs")

plugins, err := loader.NewPluginLoader(repoPath)
if err != nil {
    t.Fatal(err)
}

if err := plugins.Initialize(); err != nil {
    t.Fatal(err)
}

if err := plugins.Inject(); err != nil {
    t.Fatal(err)
}

repo, err := fsrepo.Open(repoPath)
if err != nil {
    t.Fatal(err)
}

ctx := context.Background()
node, err := core.NewNode(ctx, &node.BuildCfg{
    Online: true,
    ExtraOpts: map[string]bool{
        "pubusb": true,
    },
    Permanent: true,
    Routing:   libp2p.DHTOption,
    Repo:      repo,
})
if err != nil {
    t.Fatal(err)
}

api, err := coreapi.NewCoreAPI(node)
if err != nil {
    t.Fatal(err)
}
```

Now, you can create the driver instance.
```go
// Create a driver instance with given name. If the driver
// name exist, it will open it instead of creating a new one.
driver, err := ipfstor.NewDriver(ctx, api, driverName)
if err != nil {
    panic(err)
}
```

# Status
The package is only for personal project usage. Do not use it on any production environment.
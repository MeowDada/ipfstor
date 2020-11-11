package drive

import (
	"context"
	"io"

	orbitdb "berty.tech/go-orbit-db"
	"berty.tech/go-orbit-db/baseorbitdb"
	"berty.tech/go-orbit-db/iface"
	"github.com/ipfs/go-cid"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/meowdada/ipfstor/ipfsutil"
	"github.com/meowdada/ipfstor/options"
	"github.com/pkg/errors"
)

const (
	keyvalueStoreType = "keyvalue"
)

var (
	// ErrNoSuchKey denotes an error that indicates no such key presents.
	ErrNoSuchKey = "no such key"
)

// Instance denotes a drive instance.
type Instance interface {
	// Name denotes the human readable name of the instance.
	Name() string

	// Address denotes the orbitdb address of the instance. The address format
	// might look as below:
	//
	// /orbitdb/{driveHash}/{driveName}
	Address() string

	// Identity denotes the api using by the user who controls this instance.
	Identity() string

	// Add adds a local file to the drive instance with given key.
	Add(ctx context.Context, key, fpath string) (File, error)

	// Get gets a file with given key from the drive instance.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// List lists all existing files which matches given prefix.
	List(ctx context.Context, prefix string) (ListResult, error)

	// Remove remove the file from the drive instance.
	Remove(ctx context.Context, key string) error

	// Grant grants permission to specific user.
	Grant(ctx context.Context, keyID, permission string) error

	// Revoke revokes permission from spcific user.
	Revoke(ctx context.Context, keyID, permission string) error

	// Close closes the drive instance and save the snapshot of the drive.
	Close(ctx context.Context) error
}

// ListResult denote a data structure contains result of list operation.
type ListResult struct {
	files []File
}

// Files returns all list results.
func (lr *ListResult) Files() []File {
	return lr.files
}

// File denotes the metadata of a file which is stored in a drive instance.
type File struct {
	Key  string
	Cid  cid.Cid
	Size int64
}

// DirectOpen is similar to Open, but it simplifies the input arguements and always use
// default setting to process.
func DirectOpen(resolve string, opts ...*options.OpenDriveOptions) (Instance, error) {
	api, err := ipfsutil.NewAPI(ipfsutil.DefaultAPIAddress)
	if err != nil {
		return nil, err
	}
	return Open(context.Background(), api, resolve, opts...)
}

// Open opens an existing drive by local or remote address.
//
// Open will not create an new instance if the drive does not present.
// The resovle could be a human readable name (available only if once
// present at local) or a remote address.
func Open(ctx context.Context, api coreiface.CoreAPI, resolve string, opts ...*options.OpenDriveOptions) (Instance, error) {
	if len(resolve) == 0 {
		return nil, errors.New("resolve name could not be empty")
	}
	if api == nil {
		return nil, errors.New("accepts only non-nil ipfs instance")
	}

	db, err := newOrbitDB(ctx, api, opts...)
	if err != nil {
		return nil, err
	}

	kv, err := openKeyValueStore(ctx, db, resolve, opts...)
	if err != nil {
		db.Close()
		return nil, err
	}

	return newDrive(api, db, kv)
}

// Raw creates an instance by directly accepting necessary components.
func Raw(db iface.OrbitDB, kv iface.KeyValueStore) Instance {
	return &drive{
		api: db.IPFS(),
		db:  db,
		kv:  kv,
	}
}

func newOrbitDB(ctx context.Context, api coreiface.CoreAPI, opts ...*options.OpenDriveOptions) (iface.OrbitDB, error) {
	opt := options.MergeOpenDriveOptions(opts...)
	return orbitdb.NewOrbitDB(ctx, api, &baseorbitdb.NewOrbitDBOptions{
		Directory: opt.Directory,
		Logger:    opt.Logger,
	})
}

func openKeyValueStore(ctx context.Context, db orbitdb.OrbitDB, dbAddr string, opts ...*options.OpenDriveOptions) (iface.KeyValueStore, error) {
	opt := options.MergeOpenDriveOptions(opts...)
	return db.KeyValue(ctx, dbAddr, &iface.CreateDBOptions{
		Directory:        opt.Directory,
		AccessController: opt.AccessController,
	})
}

func boolPtr(flag bool) *bool {
	return &flag
}

func strPtr(str string) *string {
	return &str
}

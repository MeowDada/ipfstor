package drive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	orbitdb "berty.tech/go-orbit-db"
	"berty.tech/go-orbit-db/baseorbitdb"
	"berty.tech/go-orbit-db/iface"
	"github.com/dustin/go-humanize"
	"github.com/ipfs/go-cid"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/meowdada/ipfstor/ipfsutil"
	"github.com/meowdada/ipfstor/options"
	"github.com/meowdada/ipfstor/pkg/format"
	"github.com/pkg/errors"
)

const (
	keyvalueStoreType = "keyvalue"

	// ListMask is a bitmask to determine which value to be printed out.
	ListMask uint32 = 31

	// ListMaskKey is a bitmask to enable listing Key fields.
	ListMaskKey uint32 = 1

	// ListMaskCid is a bitmask to enable listing Cid fields.
	ListMaskCid uint32 = 2

	// ListMaskSize is a bitmask to enable listing Size fields.
	ListMaskSize uint32 = 4

	// ListMaskTime is a bitmask to enable listing Time fields.
	ListMaskTime uint32 = 8

	// ListMaskOwner is a bitmask to enable listing Owner fields.
	ListMaskOwner uint32 = 16
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

	// Stat stats a file with given key from the drive.
	Stat(ctx context.Context, key string) (File, error)

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

// WriteTo implements io.WriterTo interface. It writes formated strings
// about the ListResult.
func (lr *ListResult) WriteTo(w io.Writer) (int64, error) {
	files := lr.files

	maxLen := 0

	sort.Slice(files, func(i, j int) bool {
		return files[i].Key < files[j].Key
	})

	for i := range files {
		if len(files[i].Key) > maxLen {
			maxLen = len(files[i].Key)
		}
	}

	var ret string
	banner := "|" + strings.Repeat("-", maxLen) + "|" + strings.Repeat("-", 46) + "|" + strings.Repeat("-", 12) + "|" + "\n"

	for i := range files {
		format := "|%-" + strconv.Itoa(maxLen) + "s|%-46s|%-12s|"
		line := fmt.Sprintf(format, files[i].Key, files[i].Cid, humanize.IBytes(uint64(files[i].Size)))
		ret += line + "\n"
	}

	final := banner + ret + banner

	return io.Copy(w, bytes.NewBuffer([]byte(final)))
}

// Bytes marshals the result into bytes.
func (lr *ListResult) Bytes(mask uint32) []byte {
	files := lr.files
	rows := make([]format.Row, len(files))

	for i := range files {
		rows[i] = files[i].row(mask)
	}

	tmpl := format.Basic{}
	return tmpl.Render(rows, format.Options{Sort: true})
}

// Files returns all list results.
func (lr *ListResult) Files() []File {
	return lr.files
}

// File denotes the metadata of a file which is stored in a drive instance.
type File struct {
	Key       string
	Cid       cid.Cid
	Size      int64
	Timestamp string
	Owner     string
}

func (f *File) row(mask uint32) format.Row {
	m := mask & ListMask
	if m == 0 {
		m = ListMask
	}

	cols := []format.Col{}
	if m&ListMaskKey == ListMaskKey {
		cols = append(cols, format.Col{Key: "Key", Value: f.Key})
	}
	if m&ListMaskCid != 0 {
		cols = append(cols, format.Col{Key: "Cid", Value: f.Cid})
	}
	if m&ListMaskSize != 0 {
		cols = append(cols, format.Col{Key: "Size", Value: f.Size})
	}
	if m&ListMaskTime != 0 {
		cols = append(cols, format.Col{Key: "Timestamp", Value: f.Timestamp})
	}
	if m&ListMaskOwner != 0 {
		cols = append(cols, format.Col{Key: "Owner", Value: f.Owner})
	}

	return format.Row(cols)
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

	_ = kv.LoadFromSnapshot(ctx)
	_ = kv.Load(ctx, -1)

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
	store, err := db.Open(ctx, dbAddr, &iface.CreateDBOptions{
		Directory:        opt.Directory,
		Overwrite:        boolPtr(false),
		LocalOnly:        boolPtr(false),
		Create:           opt.Create,
		StoreType:        strPtr(keyvalueStoreType),
		AccessController: opt.AccessController,
		Replicate:        boolPtr(true),
	})
	if err != nil {
		return nil, err
	}
	return store.(baseorbitdb.KeyValueStore), nil
}

func boolPtr(flag bool) *bool {
	return &flag
}

func strPtr(str string) *string {
	return &str
}

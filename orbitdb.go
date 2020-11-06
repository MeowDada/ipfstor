package ipfstor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	orbitdb "berty.tech/go-orbit-db"
	"berty.tech/go-orbit-db/address"
	"berty.tech/go-orbit-db/baseorbitdb"
	"berty.tech/go-orbit-db/iface"
	"berty.tech/go-orbit-db/stores/basestore"
	files "github.com/ipfs/go-ipfs-files"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/meowdada/ipfstor/pkg/codec"
	"github.com/meowdada/ipfstor/pkg/object"
)

var (
	defaultDBPath = filepath.Join(os.Getenv("HOME"), ".orbitdb")

	defaultListWorkers = 4

	// ErrNoSuchKey raised when the key does not present.
	ErrNoSuchKey = errors.New("no such key")
)

// OpenDrive opens an existing drive.
func OpenDrive(ctx context.Context, api coreiface.CoreAPI, name string) (Driver, error) {
	db, err := orbitdb.NewOrbitDB(ctx, api, &baseorbitdb.NewOrbitDBOptions{
		Directory: &defaultDBPath,
	})
	if err != nil {
		return nil, err
	}

	// If the address is a human readable name.
	if err := address.IsValid(name); err != nil {
		addr, err := db.DetermineAddress(ctx, name, "keyvalue", &iface.DetermineAddressOptions{})
		if err != nil {
			return nil, err
		}
		name = addr.String()
	}

	store, err := db.Open(ctx, name, &iface.CreateDBOptions{
		Directory: &defaultDBPath,
		Create:    boolPtr(false),
		Overwrite: boolPtr(false),
		StoreType: stringPtr("keyvalue"),
	})
	if err != nil {
		db.Close()
		fmt.Println(err)
		return nil, err
	}

	kv, ok := store.(iface.KeyValueStore)
	if !ok {
		db.Close()
		return nil, fmt.Errorf("invalid database type")
	}

	// Load snapshot from local storage. If there is no existing snapshot, an
	// error message will be print out instead of crashing the program.
	if err := kv.LoadFromSnapshot(ctx); err != nil {
		fmt.Println(err)
	}

	return &drive{
		api:   api,
		db:    db,
		kv:    kv,
		codec: codec.Gob{},
	}, nil
}

// NewDriver creates an instance of Driver.
func NewDriver(ctx context.Context, api coreiface.CoreAPI, name string) (Driver, error) {
	db, err := orbitdb.NewOrbitDB(ctx, api, &baseorbitdb.NewOrbitDBOptions{
		Directory: &defaultDBPath,
	})
	if err != nil {
		return nil, err
	}

	kv, err := db.KeyValue(ctx, name, &iface.CreateDBOptions{})
	if err != nil {
		db.Close()
		return nil, err
	}

	// Load snapshot from local storage. If there is no existing snapshot, an
	// error message will be print out instead of crashing the program.
	if err := kv.LoadFromSnapshot(ctx); err != nil {
		fmt.Println(err)
	}

	return &drive{
		api:   api,
		db:    db,
		kv:    kv,
		codec: codec.Gob{},
	}, nil
}

type drive struct {
	name string

	api coreiface.CoreAPI
	db  iface.OrbitDB
	kv  iface.KeyValueStore

	codec codec.Instance
}

func (d *drive) Address() string {
	return d.kv.Address().String()
}

// List returns a result list consist of listed objects.
func (d *drive) List(ctx context.Context) ([]ListResult, error) {
	var lrs []ListResult

	maps := d.kv.All()
	for k, v := range maps {
		meta := d.mustDecodeMetadata(v)
		lrs = append(lrs, ListResult{
			Key:   k,
			Cid:   meta.Cid,
			Size:  meta.Size,
			Owner: meta.Owner,
		})
	}
	return lrs, nil
}

// Iter iterates all available objects in the driver.
//
// If the callback function is a nil pointer, this function will return
// immediately without any error.
func (d *drive) Iter(ctx context.Context, iterCb IterDriveFn) error {
	if iterCb == nil {
		return nil
	}

	maps := d.kv.All()
	for k, v := range maps {
		meta := d.mustDecodeMetadata(v)
		if err := iterCb(ctx, k, meta.Cid, meta.Size, meta.Owner); err != nil {
			return err
		}
	}
	return nil
}

// Add adds a file with given key to the driver.
//
// It will first put the file to IPFS then update the
// key value store.
func (d *drive) Add(ctx context.Context, key, path string) error {
	unixfs := d.api.Unixfs()

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	node := files.NewReaderStatFile(f, info)

	r, err := unixfs.Add(ctx, node)
	if err != nil {
		return err
	}

	val := d.mustEncode(object.Metadata{
		Cid:   r.Cid(),
		Size:  info.Size(),
		Owner: d.db.Identity().ID,
	})

	_, err = d.kv.Put(ctx, key, val)
	return err
}

// Get returns the file reader with key. If the file does not exist or
// be no longer available. A specifc error ErrNoSuchKey will be returned.
func (d *drive) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	unixfs := d.api.Unixfs()

	meta, err := d.getMetaByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	p := path.IpfsPath(meta.Cid)

	node, err := unixfs.Get(ctx, p)
	if err != nil {
		return nil, err
	}

	return files.ToFile(node), nil
}

// Remove removes the data from the driver. Once the key has been removed,
// the data will be invisible from the driver, but the actual data will not be
// removed immediately. It will be removed only when it becomes the victims of
// ipfs garbage collection.
func (d *drive) Remove(ctx context.Context, key string) error {
	data, err := d.kv.Get(ctx, key)
	if err != nil {
		return err
	}

	meta := d.mustDecodeMetadata(data)
	p := path.IpfsPath(meta.Cid)

	// Unpin the data.
	if err := d.api.Pin().Rm(ctx, p); err != nil {
		return err
	}

	_, err = d.kv.Delete(ctx, key)
	return err
}

// Close saves current snapshot and closes all underlying modules.
func (d *drive) Close(ctx context.Context) error {
	if d.db == nil {
		return nil
	}

	// Save the snapshot to local cache before exiting.
	if _, err := basestore.SaveSnapshot(ctx, d.kv); err != nil {
		log.Println(err)
	}

	return d.db.Close()
}

func (d *drive) mustEncode(v interface{}) []byte {
	data, _ := d.codec.Marshal(v)
	return data
}

func (d *drive) mustDecode(data []byte, v interface{}) error {
	return d.codec.Unmarshal(data, v)
}

func (d *drive) mustDecodeMetadata(data []byte) object.Metadata {
	var meta object.Metadata
	_ = d.codec.Unmarshal(data, &meta)
	return meta
}

func (d *drive) getMetaByKey(ctx context.Context, key string) (object.Metadata, error) {
	data, err := d.kv.Get(ctx, key)
	if err != nil {
		return object.Metadata{}, err
	}
	if data == nil && err == nil {
		return object.Metadata{}, ErrNoSuchKey
	}
	return d.mustDecodeMetadata(data), nil
}

func boolPtr(flag bool) *bool {
	return &flag
}

func stringPtr(str string) *string {
	return &str
}

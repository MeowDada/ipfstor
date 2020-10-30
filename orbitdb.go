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

	// ErrNoSuchKey raised when the key does not present.
	ErrNoSuchKey = errors.New("no such key")
)

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
func (d *drive) Add(ctx context.Context, key string, reader io.Reader) error {
	f := files.NewReaderFile(reader)
	unixfs := d.api.Unixfs()

	resolved, err := unixfs.Add(ctx, f)
	if err != nil {
		return err
	}

	size, err := f.Size()
	if err != nil {
		return err
	}

	val := d.mustEncode(object.Metadata{
		Cid:   resolved.Cid(),
		Size:  size,
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

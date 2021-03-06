package drive

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"berty.tech/go-orbit-db/iface"
	"berty.tech/go-orbit-db/stores/basestore"
	files "github.com/ipfs/go-ipfs-files"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/meowdada/ipfstor/pkg/codec"
)

type drive struct {
	api coreiface.CoreAPI
	db  iface.OrbitDB
	kv  iface.KeyValueStore
}

func (d *drive) Name() string {
	return d.kv.DBName()
}

func (d *drive) Address() string {
	return d.kv.Address().String()
}

func (d *drive) Identity() string {
	return d.kv.Identity().ID
}

func (d *drive) AddFile(ctx context.Context, key, fpath string) (File, error) {
	if len(fpath) == 0 || len(key) == 0 {
		return File{}, fmt.Errorf("Either key or fpath cannot be empty string")
	}

	node, err := openFileNode(fpath)
	if err != nil {
		return File{}, err
	}

	size, err := node.Size()
	if err != nil {
		return File{}, err
	}

	unixfs := d.api.Unixfs()
	unixfsOpts := options.Unixfs.
		Pin(true)

	resolve, err := unixfs.Add(ctx, node, unixfsOpts)
	if err != nil {
		return File{}, err
	}

	f := File{
		Key:       key,
		Cid:       resolve.Cid(),
		Size:      size,
		Timestamp: time.Now().Format(time.RFC1123),
		Owner:     d.Identity(),
	}

	data := mustEncodeGob(f)

	_, err = d.kv.Put(ctx, key, data)
	if err != nil {
		return File{}, err
	}

	return f, nil
}

func (d *drive) Add(ctx context.Context, key string, r io.Reader) (File, error) {
	if len(key) == 0 {
		return File{}, fmt.Errorf("cannot use empty key")
	}
	if r == nil {
		return File{}, fmt.Errorf("input stream is a nil pointer")
	}

	node := newFile(key, r)

	unixfs := d.api.Unixfs()
	unixfsOpts := options.Unixfs.
		Pin(true)

	resolve, err := unixfs.Add(ctx, node, unixfsOpts)
	if err != nil {
		return File{}, err
	}

	size, err := node.Size()
	if err != nil {
		return File{}, err
	}

	f := File{
		Key:       key,
		Cid:       resolve.Cid(),
		Size:      size,
		Timestamp: time.Now().Format(time.RFC1123),
		Owner:     d.Identity(),
	}

	data := mustEncodeGob(f)

	_, err = d.kv.Put(ctx, key, data)
	if err != nil {
		return File{}, err
	}

	return f, nil
}

func (d *drive) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("cannot use empty key")
	}

	data, err := d.kv.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	f := mustDecodeGob(data)
	addr := f.Cid
	unixfs := d.api.Unixfs()

	node, err := unixfs.Get(ctx, path.IpfsPath(addr))
	if err != nil {
		return nil, err
	}
	return files.ToFile(node), nil
}

func (d *drive) Stat(ctx context.Context, key string) (File, error) {
	if len(key) == 0 {
		return File{}, ErrEmptyKey
	}

	data, err := d.kv.Get(ctx, key)
	if err != nil {
		return File{}, err
	}
	if data == nil {
		return File{}, ErrNoSuchKey
	}

	return decodeGob(data)
}

func (d *drive) List(ctx context.Context, prefix string) (ListResult, error) {
	vals := d.kv.All()

	var files []File
	for k, v := range vals {
		if strings.Contains(k, prefix) {
			f := mustDecodeGob(v)
			files = append(files, f)
		}
	}

	return ListResult{
		files: files,
	}, nil
}

func (d *drive) Remove(ctx context.Context, key string) error {
	data, err := d.kv.Get(ctx, key)
	if err != nil {
		return err
	}

	f := mustDecodeGob(data)

	pin := d.api.Pin()

	_, ok, err := pin.IsPinned(ctx, path.IpfsPath(f.Cid))
	if err != nil {
		return err
	}

	if ok {
		if err := pin.Rm(ctx, path.IpfsPath(f.Cid), options.Pin.RmRecursive(true)); err != nil {
			return err
		}
	}

	_, err = d.kv.Delete(ctx, key)
	return err
}

func (d *drive) Grant(ctx context.Context, keyID, permission string) error {
	ac := d.kv.AccessController()
	return ac.Grant(ctx, permission, keyID)
}

func (d *drive) Revoke(ctx context.Context, keyID, permission string) error {
	ac := d.kv.AccessController()
	return ac.Revoke(ctx, permission, keyID)
}

func (d *drive) Close(ctx context.Context) error {
	// Save snapshopt.
	_, err := basestore.SaveSnapshot(ctx, d.kv)
	if err != nil {
		return err
	}

	if err := d.kv.Close(); err != nil {
		return err
	}

	if err := d.db.Close(); err != nil {
		return err
	}

	return nil
}

func newDrive(api coreiface.CoreAPI, db iface.OrbitDB, kv iface.KeyValueStore) (*drive, error) {
	return &drive{
		api: api,
		db:  db,
		kv:  kv,
	}, nil
}

func openFileNode(fpath string) (files.Node, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	node := files.NewReaderStatFile(f, info)
	return node, nil
}

func mustEncodeGob(v interface{}) []byte {
	encoder := codec.Gob{}
	data, _ := encoder.Marshal(v)
	return data
}

func mustDecodeGob(data []byte) (f File) {
	decoder := codec.Gob{}
	decoder.Unmarshal(data, &f)
	return f
}

func decodeGob(data []byte) (f File, err error) {
	decoder := codec.Gob{}
	err = decoder.Unmarshal(data, &f)
	return f, err
}

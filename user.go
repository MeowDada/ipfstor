package ipfstor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

// NewUser creates an instance of User.
func NewUser(api coreiface.CoreAPI) User {
	return &user{
		api: api,
	}
}

type user struct {
	api coreiface.CoreAPI
}

func (u *user) GenerateKeyFile(ctx context.Context, path string) error {
	k, err := generateSwarmKey()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("/key/swarm/psk/1.0.0/\n")
	if err != nil {
		return err
	}

	_, err = f.WriteString("/base16/\n")
	if err != nil {
		return err
	}

	_, err = f.WriteString(hex.EncodeToString(k))
	return err
}

func (u *user) Key(ctx context.Context) (path.Path, error) {
	k, err := u.api.Key().Self(ctx)
	if err != nil {
		return nil, err
	}
	return k.Path(), nil
}

func (u *user) AddPeer(ctx context.Context, addr string) error {
	baseURL, err := url.Parse("http://127.0.0.1:5001/api/v0/bootstrap/add")
	if err != nil {
		return err
	}

	params := url.Values{}
	params.Add("arg", addr)

	baseURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL.String(), nil)
	if err != nil {
		return err
	}
	fmt.Println(req.URL)

	clnt := http.Client{}
	resp, err := clnt.Do(req)
	if err != nil {
		return err
	}

	msg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(msg))

	return nil
}

func generateSwarmKey() ([]byte, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return b, err
}

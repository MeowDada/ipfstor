package codec

import (
	"bytes"
	"encoding/gob"
)

// Gob uses gobinary as codec backend.
type Gob struct{}

const gobName = "gob"

// Name denotes the algorithm used by the codec instance.
func (g Gob) Name() string {
	return gobName
}

// Marshal encodes input data structure into go binaries.
func (g Gob) Marshal(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(v)
	return b.Bytes(), err
}

// Unmarshal decodes input byte array and populates fields of input data
// structure.
func (g Gob) Unmarshal(data []byte, v interface{}) error {
	b := bytes.NewBuffer(data)
	dec := gob.NewDecoder(b)
	return dec.Decode(v)
}

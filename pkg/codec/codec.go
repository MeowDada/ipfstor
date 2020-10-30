package codec

// Instance provides both encoder and decoder methods.
type Instance interface {
	// Name denotes the algorithm used by the codec instance.
	Name() string

	// Marshal marshals given data structure into a byte array.
	Marshal(v interface{}) ([]byte, error)

	// Unmarshal decodes input byte array and populates the input
	// data structure. If either input byte array or data structure
	// is a nil pointer, a specific error will be returned.
	Unmarshal(data []byte, v interface{}) error
}

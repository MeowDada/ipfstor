package options

// OpenDriveOptions configures behaviour while opening a drive.
type OpenDriveOptions struct {
	Directory *string
}

// Directory sets the Directory field of the OpenDriveOptions.
func (o *OpenDriveOptions) SetDirectory(dir string) *OpenDriveOptions {
	return &OpenDriveOptions{
		Directory: &dir,
	}
}

// OpenDrive creates a new OpenDriveOptions instance.
func OpenDrive() *OpenDriveOptions {
	return &OpenDriveOptions{}
}

// MergeOpenDriveOptions combines given OpenDriveOptions into a single OpenDriveOption in
// a last-one-wins fashion.
func MergeOpenDriveOptions(opts ...*OpenDriveOptions) *OpenDriveOptions {
	o := OpenDrive()

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.Directory != nil {
			o.Directory = opt.Directory
		}
	}

	return o
}

package options

import (
	"berty.tech/go-orbit-db/accesscontroller"
	"go.uber.org/zap"
)

// OpenDriveOptions configures behaviour while opening a drive.
type OpenDriveOptions struct {
	Directory        *string
	Logger           *zap.Logger
	AccessController accesscontroller.ManifestParams
	Create           *bool
}

// SetDirectory sets the Directory field of the OpenDriveOptions.
func (o *OpenDriveOptions) SetDirectory(dir string) *OpenDriveOptions {
	o.Directory = &dir
	return o
}

// SetLogger sets the Logger field of the OpenDriveOptions.
func (o *OpenDriveOptions) SetLogger(logger *zap.Logger) *OpenDriveOptions {
	o.Logger = logger
	return o
}

// SetAccessController sets the AccessController field of the OpenDriveOptions.
func (o *OpenDriveOptions) SetAccessController(controller accesscontroller.ManifestParams) *OpenDriveOptions {
	o.AccessController = controller
	return o
}

// SetCreate sets the Create field of the OpenDriveOptions.
func (o *OpenDriveOptions) SetCreate(flag bool) *OpenDriveOptions {
	o.Create = &flag
	return o
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
		if opt.Logger != nil {
			o.Logger = opt.Logger
		}
		if opt.AccessController != nil {
			o.AccessController = opt.AccessController
		}
		if opt.Create != nil {
			o.Create = opt.Create
		}
	}

	return o
}

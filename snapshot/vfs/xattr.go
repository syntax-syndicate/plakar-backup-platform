package vfs

import (
	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/vmihailenco/msgpack/v5"
)

const VFS_XATTR_VERSION = "1.0.0"

func init() {
	versioning.Register(resources.RT_XATTR_BTREE, versioning.FromString(btree.BTREE_VERSION))
	versioning.Register(resources.RT_XATTR_ENTRY, versioning.FromString(VFS_XATTR_VERSION))
}

type Xattr struct {
	Version versioning.Version `msgpack:"version" json:"version"`
	Name    string             `msgpack:"name" json:"name"`
	Size    int64              `msgpack:"size" json:"size"`
	Object  objects.MAC        `msgpack:"object,omitempty" json:"-"`

	// This the true object, resolved when opening the
	// xattr. Beware we serialize it as "Object" only for json to
	// not break API compat.
	ResolvedObject *objects.Object `msgpack:"-" json:"object,omitempty"`
}

func NewXattr(record importer.ScanRecord, object *objects.Object) *Xattr {
	var size int64

	for _, chunk := range object.Chunks {
		size += int64(chunk.Length)
	}

	return &Xattr{
		Version: versioning.FromString(VFS_XATTR_VERSION),
		Name:    record.FileInfo.Lname,
		Object:  object.MAC,
		Size:    size,
	}
}

func XattrFromBytes(bytes []byte) (*Xattr, error) {
	xattr := &Xattr{}
	err := msgpack.Unmarshal(bytes, &xattr)
	return xattr, err
}

func (x *Xattr) ToBytes() ([]byte, error) {
	return msgpack.Marshal(x)
}

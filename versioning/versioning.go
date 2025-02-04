package versioning

import (
	"fmt"
	"sync"

	"github.com/PlakarKorp/plakar/resources"
)

type Version uint32

func NewVersion(major, minor, patch uint32) Version {
	return Version(major<<16 | minor<<8 | patch)
}

func (v Version) Major() uint32 {
	return uint32(v >> 16 & 0xff)
}

func (v Version) Minor() uint32 {
	return uint32(v >> 8 & 0xff)
}

func (v Version) Patch() uint32 {
	return uint32(v & 0xff)
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major(), v.Minor(), v.Patch())
}

func FromString(s string) Version {
	var major, minor, patch uint32
	_, err := fmt.Sscanf(s, "%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		panic(err)
	}
	return NewVersion(major, minor, patch)
}

var currentVersions map[resources.Type]Version = make(map[resources.Type]Version)
var currentVersionsMu sync.Mutex

func RegisterVersion(resourceType resources.Type, version Version) {
	currentVersionsMu.Lock()
	defer currentVersionsMu.Unlock()
	currentVersions[resourceType] = version
}

func CurrentVersion(resourceType resources.Type) Version {
	currentVersionsMu.Lock()
	defer currentVersionsMu.Unlock()

	fmt.Println("resourceType", resourceType)
	ret, found := currentVersions[resourceType]
	if !found {
		panic("version not found for resource type" + string(resourceType))
	}
	return ret
}

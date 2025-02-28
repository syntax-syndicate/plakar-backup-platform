package repository

import (
	"io"
	"time"

	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/vmihailenco/msgpack/v5"
)

const LOCK_VERSION = "1.0.0"

func init() {
	versioning.Register(resources.RT_LOCK, versioning.FromString(LOCK_VERSION))
}

type Lock struct {
	Version   versioning.Version `msgpack:"version"`
	Timestamp time.Time          `msgpack:"timestamp"`
	Hostname  string             `msgpack:"hostname"`
	Exclusive bool               `msgpack:"exclusive"`
}

func newLock(hostname string, exclusive bool) *Lock {
	return &Lock{
		Timestamp: time.Now(),
		Hostname:  hostname,
		Exclusive: exclusive,
	}
}

func NewExclusiveLock(hostname string) *Lock {
	return newLock(hostname, true)
}

func NewSharedLock(hostname string) *Lock {
	return newLock(hostname, false)
}

func NewLockFromStream(version versioning.Version, rd io.Reader) (*Lock, error) {
	var lock Lock
	if err := msgpack.NewDecoder(rd).Decode(&lock); err != nil {
		return nil, err
	}

	return &lock, nil
}

func (lock *Lock) SerializeToStream(w io.Writer) error {
	return msgpack.NewEncoder(w).Encode(lock)
}

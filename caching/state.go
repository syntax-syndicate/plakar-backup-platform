package caching

import (
	"iter"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
)

type StateCache interface {
	PutState(stateID objects.MAC, data []byte) error
	HasState(stateID objects.MAC) (bool, error)
	GetState(stateID objects.MAC) ([]byte, error)
	DelState(stateID objects.MAC) error
	GetStates() (map[objects.MAC][]byte, error)

	PutDelta(blobType resources.Type, blobCsum objects.MAC, data []byte) error
	GetDelta(blobType resources.Type, blobCsum objects.MAC) ([]byte, error)
	HasDelta(blobType resources.Type, blobCsum objects.MAC) (bool, error)
	GetDeltaByCsum(blobCsum objects.MAC) ([]byte, error)
	GetDeltasByType(blobType resources.Type) iter.Seq2[objects.MAC, []byte]
	GetDeltas() iter.Seq2[objects.MAC, []byte]

	PutDeleted(blobType resources.Type, blobCsum objects.MAC, data []byte) error
	HasDeleted(blobType resources.Type, blobCsum objects.MAC) (bool, error)
	GetDeleteds() iter.Seq2[objects.MAC, []byte]

	PutPackfile(stateID objects.MAC, packfile objects.MAC, data []byte) error
	GetPackfiles() iter.Seq2[objects.MAC, []byte]
	GetPackfilesForState(stateID objects.MAC) iter.Seq2[objects.MAC, []byte]
}

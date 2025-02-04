package snapshot

import (
	"iter"
	"strings"

	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
)

const ERROR_VERSION = "1.0.0"

func init() {
	versioning.Register(resources.RT_ERROR, versioning.FromString(ERROR_VERSION))
}

type ErrorItem struct {
	Version versioning.Version `msgpack:"version" json:"version"`
	Name    string             `msgpack:"name" json:"name"`
	Error   string             `msgpack:"error" json:"error"`
}

func (snapshot *Snapshot) Errors(beneath string) (iter.Seq2[ErrorItem, error], error) {
	if !strings.HasSuffix(beneath, "/") {
		beneath += "/"
	}

	rd, err := snapshot.repository.GetBlob(resources.RT_ERROR, snapshot.Header.GetSource(0).Errors)
	if err != nil {
		return nil, err
	}

	storage := SnapshotStore[string, ErrorItem]{
		blobtype: resources.RT_ERROR,
		snap:     snapshot,
	}
	tree, err := btree.Deserialize(rd, &storage, strings.Compare)
	if err != nil {
		return nil, err
	}

	return func(yield func(ErrorItem, error) bool) {
		iter, err := tree.ScanFrom(beneath)
		if err != nil {
			yield(ErrorItem{}, err)
			return
		}

		for iter.Next() {
			_, item := iter.Current()
			if !strings.HasPrefix(item.Name, beneath) {
				break
			}
			if !yield(item, nil) {
				break
			}
		}
		if err := iter.Err(); err != nil {
			yield(ErrorItem{}, err)
			return
		}
	}, nil
}

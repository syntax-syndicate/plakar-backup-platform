package snapshot

import (
	"iter"
	"strings"

	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/vmihailenco/msgpack/v5"
)

type ErrorItem struct {
	Name  string `msgpack:"name" json:"name"`
	Error string `msgpack:"error" json:"error"`
}

func (snapshot *Snapshot) Errors(beneath string) (iter.Seq2[ErrorItem, error], error) {
	if !strings.HasSuffix(beneath, "/") {
		beneath += "/"
	}

	bytes, err := snapshot.GetBlob(packfile.TYPE_ERROR, snapshot.Header.Errors)
	if err != nil {
		return nil, err
	}

	var root btree.BTree[string, objects.Checksum, ErrorItem]
	if err := msgpack.Unmarshal(bytes, &root); err != nil {
		return nil, err
	}

	storage := SnapshotStore[string, ErrorItem]{
		blobtype: packfile.TYPE_ERROR,
		snap:     snapshot,
	}
	tree := btree.FromStorage(root.Root, &storage, strings.Compare, root.Order)

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

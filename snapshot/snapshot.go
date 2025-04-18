package snapshot

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("snapshot not found")
)

type Snapshot struct {
	repository *repository.Repository
	checkCache *caching.CheckCache

	filesystem *vfs.Filesystem

	Header *header.Header
}

func LogicalSize(repo *repository.Repository) (int, int64, error) {
	nSnapshots := 0
	totalSize := int64(0)
	for snapshotID := range repo.ListSnapshots() {
		snap, err := Load(repo, snapshotID)
		if err != nil {
			continue
		}
		nSnapshots++
		totalSize += int64(snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size)
		snap.Close()
	}

	return nSnapshots, totalSize, nil
}

func Load(repo *repository.Repository, Identifier objects.MAC) (*Snapshot, error) {
	hdr, _, err := GetSnapshot(repo, Identifier)
	if err != nil {
		return nil, err
	}

	snapshot := &Snapshot{}
	snapshot.repository = repo
	snapshot.Header = hdr

	repo.Logger().Trace("snapshot", "%x: Load()", snapshot.Header.GetIndexShortID())
	return snapshot, nil
}

func (snap *Snapshot) Close() error {
	snap.Logger().Trace("snapshot", "%x: Close(): %x", snap.Header.Identifier, snap.Header.GetIndexShortID())

	return nil
}

func (snap *Snapshot) AppContext() *appcontext.AppContext {
	return snap.repository.AppContext()
}

func (snap *Snapshot) Event(evt events.Event) {
	snap.AppContext().Events().Send(evt)
}

func GetSnapshot(repo *repository.Repository, Identifier objects.MAC) (*header.Header, bool, error) {
	repo.Logger().Trace("snapshot", "repository.GetSnapshot(%x)", Identifier)

	// Try to get snapshot from cache first
	cache, err := repo.AppContext().GetCache().Repository(repo.Configuration().RepositoryID)
	if err == nil {
		if snapshotBytes, err := cache.GetSnapshot(Identifier); err == nil {
			if snapshotBytes != nil {
				hdr, err := header.NewFromBytes(snapshotBytes)
				if err == nil {
					return hdr, true, nil
				}
			}
		}
	}

	rd, err := repo.GetBlob(resources.RT_SNAPSHOT, Identifier)
	if err != nil {
		if errors.Is(err, repository.ErrBlobNotFound) {
			err = ErrNotFound
		}
		return nil, false, err
	}

	buffer, err := io.ReadAll(rd)
	if err != nil {
		return nil, false, err
	}

	hdr, err := header.NewFromBytes(buffer)
	if err != nil {
		return nil, false, err
	}

	if cache != nil {
		_ = cache.PutSnapshot(Identifier, buffer) // optionally handle/log error
	}

	return hdr, false, nil
}

func (snap *Snapshot) LookupObject(mac objects.MAC) (*objects.Object, error) {
	buffer, err := snap.repository.GetBlobBytes(resources.RT_OBJECT, mac)
	if err != nil {
		return nil, err
	}
	return objects.NewObjectFromBytes(buffer)
}

func getPackfileForBlobWithError(snap *Snapshot, res resources.Type, mac objects.MAC) (objects.MAC, error) {
	packfile, exists, err := snap.repository.GetPackfileForBlob(res, mac)
	if err != nil {
		return objects.MAC{}, fmt.Errorf("Error %s while trying to locate packfile for blob %x of type %s", err, mac, res)
	} else if !exists {
		return objects.MAC{}, fmt.Errorf("Could not find packfile for blob %x of type %s", mac, res)
	} else {
		return packfile, nil
	}
}

func (snap *Snapshot) ListPackfiles() iter.Seq2[objects.MAC, error] {
	return func(yield func(objects.MAC, error) bool) {
		var err error
		for res, mac := range snap.MACs(&err) {
			if !yield(getPackfileForBlobWithError(snap, res, mac)) {
				return
			}
		}
		if err != nil {
			yield(objects.MAC{}, err)
		}
	}
}

func (snap *Snapshot) MACs(ret *error) iter.Seq2[resources.Type, objects.MAC] {
	macidx, rootmac, err := snap.MACIdx()
	if err != nil {
		*ret = err
		return func(func(resources.Type, objects.MAC) bool) {}
	}

	if macidx == nil {
		return snap.CrawlMACs(ret)
	}

	it := macidx.IterDFS()
	return func(yield func(resources.Type, objects.MAC) bool) {
		if !yield(resources.RT_SNAPSHOT, snap.Header.Identifier) {
			return
		}

		if snap.Header.Identity.Identifier != uuid.Nil {
			if !yield(resources.RT_SIGNATURE, snap.Header.Identifier) {
				return
			}
		}

		if !yield(resources.RT_BTREE_ROOT, rootmac) {
			return
		}

		for it.Next() {
			mac, node := it.Current()
			if !yield(resources.RT_BTREE_NODE, mac) {
				return
			}
			if len(node.Values) != 0 {
				for _, tup := range node.Keys {
					if !yield(tup.Resource, tup.MAC) {
						return
					}
				}
			}
		}
	}
}

func (snap *Snapshot) CrawlMACs(ret *error) iter.Seq2[resources.Type, objects.MAC] {
	pvfs, err := snap.Filesystem()
	return func(yield func(resources.Type, objects.MAC) bool) {
		if err != nil {
			*ret = err
		}

		if !yield(resources.RT_SNAPSHOT, snap.Header.Identifier) {
			return
		}

		if snap.Header.Identity.Identifier != uuid.Nil {
			if !yield(resources.RT_SIGNATURE, snap.Header.Identifier) {
				return
			}
		}

		if !yield(resources.RT_VFS_BTREE, snap.Header.Sources[0].VFS.Root) {
			return
		}

		/* Iterate over all the VFS, resolving both Nodes and actual VFS entries. */
		fsIter := pvfs.IterNodes()
		for fsIter.Next() {
			macNode, node := fsIter.Current()
			if !yield(resources.RT_VFS_NODE, macNode) {
				return
			}

			for _, entry := range node.Values {
				if !yield(resources.RT_VFS_ENTRY, entry) {
					return
				}

				vfsEntry, err := pvfs.ResolveEntry(entry)
				if err != nil {
					*ret = fmt.Errorf("Failed to resolve entry %x: %v", entry, err)
					return
				}

				if vfsEntry.HasObject() {
					if !yield(resources.RT_OBJECT, vfsEntry.Object) {
						return
					}

					for _, chunk := range vfsEntry.ResolvedObject.Chunks {
						if !yield(resources.RT_CHUNK, chunk.ContentMAC) {
							return
						}
					}
				}
			}
		}
		if err := fsIter.Err(); err != nil {
			*ret = err
			return
		}

		if !yield(resources.RT_ERROR_BTREE, snap.Header.Sources[0].VFS.Errors) {
			return
		}
		errIter := pvfs.IterErrorNodes()
		for errIter.Next() {
			macNode, node := errIter.Current()
			if !yield(resources.RT_ERROR_NODE, macNode) {
				return
			}

			for _, error := range node.Values {
				if !yield(resources.RT_ERROR_ENTRY, error) {
					return
				}
			}
		}
		if err := errIter.Err(); err != nil {
			*ret = err
			return
		}

		if !yield(resources.RT_XATTR_BTREE, snap.Header.Sources[0].VFS.Xattrs) {
			return
		}
		xattrIter := pvfs.XattrNodes()
		for xattrIter.Next() {
			mac, node := xattrIter.Current()
			if !yield(resources.RT_XATTR_NODE, mac) {
				return
			}

			for _, error := range node.Values {
				if !yield(resources.RT_XATTR_ENTRY, error) {
					return
				}
			}
		}
		if err := xattrIter.Err(); err != nil {
			*ret = err
			return
		}

		// Lastly going over the indexes.
		if !yield(resources.RT_BTREE_ROOT, snap.Header.GetSource(0).Indexes[0].Value) {
			return
		}
		rd, err := snap.repository.GetBlob(resources.RT_BTREE_ROOT, snap.Header.GetSource(0).Indexes[0].Value)
		if err != nil {
			*ret = fmt.Errorf("Failed to load Index root entry: %v", err)
			return
		}

		store := repository.NewRepositoryStore[string, objects.MAC](snap.repository, resources.RT_BTREE_NODE)
		tree, err := btree.Deserialize(rd, store, strings.Compare)
		if err != nil {
			*ret = fmt.Errorf("Failed to deserialize root entry: %v", err)
			return
		}

		indexIter := tree.IterDFS()
		for indexIter.Next() {
			mac, _ := indexIter.Current()
			if !yield(resources.RT_BTREE_NODE, mac) {
				return
			}
		}
		if err := indexIter.Err(); err != nil {
			*ret = err
			return
		}

		// cover also the mac index
		root := snap.Header.GetSource(0).Indexes[1].Value
		if !yield(resources.RT_BTREE_ROOT, root) {
			return
		}

		rd, err = snap.repository.GetBlob(resources.RT_BTREE_ROOT, root)
		if err != nil {
			*ret = err
			return
		}

		macstore := repository.NewRepositoryStore[objects.MACTuple, struct{}](snap.repository,
			resources.RT_BTREE_NODE)
		mactree, err := btree.Deserialize(rd, macstore, objects.MACTupleCompare)
		if err != nil {
			*ret = err
			return
		}

		macIter := mactree.IterDFS()
		for macIter.Next() {
			mac, _ := macIter.Current()
			if !yield(resources.RT_BTREE_NODE, mac) {
				return
			}
		}
		if err := macIter.Err(); err != nil {
			*ret = err
			return
		}
	}
}

func (snap *Snapshot) Logger() *logging.Logger {
	return snap.AppContext().GetLogger()
}

func (snap *Snapshot) SetCheckCache(cache *caching.CheckCache) {
	snap.checkCache = cache
}

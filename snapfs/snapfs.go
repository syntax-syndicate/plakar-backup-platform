package snapfs

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/PlakarKorp/plakar/caching/lru"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
)

type snapfs struct {
	repo  *repository.Repository
	cache *lru.Cache[objects.MAC, *vfs.Filesystem]
}

func NewFS(repo *repository.Repository) (FS, error) {
	return &snapfs{
		repo:  repo,
		cache: lru.New[objects.MAC, *vfs.Filesystem](30, nil),
	}, nil
}

func (sfs *snapfs) loadvfs(snapid string) (*vfs.Filesystem, error) {
	id, err := hex.DecodeString(snapid)
	if err != nil {
		return nil, err
	}
	if len(id) != 32 {
		return nil, fmt.Errorf("invalid snapshot name")
	}

	vfs, ok := sfs.cache.Get(objects.MAC(id))
	if ok {
		return vfs, nil
	}

	snap, err := snapshot.Load(sfs.repo, objects.MAC(id))
	if err != nil {
		return nil, err
	}

	vfs, err = snap.Filesystem()
	if err != nil {
		return nil, err
	}

	return vfs.Mount("/" + hex.EncodeToString(id[:]))
}

func splitpath(name string) (snapid, snappath string, err error) {
	name = path.Clean(name)
	if !path.IsAbs(name) {
		return "", "", fs.ErrInvalid
	}

	if name == "/" {
		return "/", "", nil
	}

	name = name[1:]
	if off := strings.Index(name, "/"); off == -1 {
		return name, name, nil
	} else {
		return name[:off], name, nil
	}
}

func (sfs *snapfs) Open(name string) (File, error) {
	snapid, snappath, err := splitpath(name)
	if err != nil {
		return nil, fs.ErrInvalid
	}

	if snapid == "/" {
		sfs.repo.RebuildState()
		return &rootdir{sfs: sfs}, nil
	}

	vfs, err := sfs.loadvfs(snapid)
	if err != nil {
		return nil, fmt.Errorf("%w: can't open snapshot: %v",
			fs.ErrInvalid, err)
	}

	entry, err := vfs.GetEntry(snappath)
	if err != nil {
		return nil, err
	}

	return entry.Open(vfs).(File), nil // impossible cast!
}

func (sfs *snapfs) Stat(name string) (FileInfo, error) {
	snapid, snappath, err := splitpath(name)
	if err != nil {
		return nil, err
	}
	if snapid == "/" {
		return &rootinfo{}, nil
	}
	vfs, err := sfs.loadvfs(snapid)
	if err != nil {
		return nil, err
	}
	return vfs.Stat(snappath)
}

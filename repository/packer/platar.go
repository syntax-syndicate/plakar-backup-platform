package packer

import (
	"context"
	"fmt"
	"hash"
	"io"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
)

type platarPackerManager struct {
	packingCache *caching.PackingCache

	packerChan     chan interface{}
	packerChanDone chan struct{}

	storageConf  *storage.Configuration
	encodingFunc func(io.Reader) (io.Reader, error)
	hashFactory  func() hash.Hash
	appCtx       *appcontext.AppContext

	// XXX: Temporary hack callback-based to ease the transition diff.
	// To be revisited with either an interface or moving this file inside repository/
	flush func(*PackWriter) error
}

func NewPlatarPackerManager(ctx *appcontext.AppContext, storageConfiguration *storage.Configuration, encodingFunc func(io.Reader) (io.Reader, error), hashFactory func() hash.Hash, flusher func(*PackWriter) error) (PackerManagerInt, error) {
	cache, err := ctx.GetCache().Packing()
	if err != nil {
		return nil, err
	}

	return &platarPackerManager{
		packingCache:   cache,
		packerChan:     make(chan interface{}, runtime.NumCPU()*2+1),
		packerChanDone: make(chan struct{}),
		storageConf:    storageConfiguration,
		encodingFunc:   encodingFunc,
		hashFactory:    hashFactory,
		appCtx:         ctx,
		flush:          flusher,
	}, nil
}

func (mgr *platarPackerManager) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pfile := NewPackWriter(mgr.flush, mgr.encodingFunc, mgr.hashFactory, mgr.packingCache)
	workerGroup, workerCtx := errgroup.WithContext(ctx)
	for i := 0; i < 1; i++ {
		workerGroup.Go(func() error {
			for {
				select {
				case <-workerCtx.Done():
					return workerCtx.Err()
				case msg, ok := <-mgr.packerChan:
					if !ok {
						return nil
					}

					pm, ok := msg.(*PackerMsg)
					if !ok {
						return fmt.Errorf("unexpected message type")
					}

					if err := pfile.WriteBlob(pm.Type, pm.Version, pm.MAC, pm.Data, pm.Flags); err != nil {
						return fmt.Errorf("failed to write blob: %w", err)
					}
				}
			}
		})
	}

	// Wait for workers to finish.
	if err := workerGroup.Wait(); err != nil {
		mgr.appCtx.GetLogger().Error("Worker group error: %s", err)
		cancel() // Propagate cancellation.
	}

	err := pfile.Finalize()
	if err != nil {
		return fmt.Errorf("failed to write packfile: %w", err)
	}

	// Signal completion.
	mgr.packerChanDone <- struct{}{}
	close(mgr.packerChanDone)

	mgr.packingCache.Close()

	return nil
}

func (mgr *platarPackerManager) Wait() {
	close(mgr.packerChan)
	<-mgr.packerChanDone
}

func (mgr *platarPackerManager) InsertIfNotPresent(Type resources.Type, mac objects.MAC) (bool, error) {
	// XXX: This is not atomic, as such it leaves a possibility of missed dedup (unlikely though). Needs to be fixed by using a batch.
	has, err := mgr.packingCache.HasBlob(Type, mac)
	if err != nil {
		return false, err
	}

	if has {
		return true, nil
	}

	if err := mgr.packingCache.PutBlob(Type, mac); err != nil {
		return false, err
	}

	return false, nil
}

func (mgr *platarPackerManager) Put(Type resources.Type, mac objects.MAC, data []byte) error {
	mgr.packerChan <- &PackerMsg{Type: Type, Version: versioning.GetCurrentVersion(Type), Timestamp: time.Now(), MAC: mac, Data: data}
	return nil
}

func (mgr *platarPackerManager) Exists(Type resources.Type, mac objects.MAC) (bool, error) {
	return mgr.packingCache.HasBlob(Type, mac)
}

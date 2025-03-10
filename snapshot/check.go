package snapshot

import (
	"bytes"
	"sync"

	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
)

type CheckOptions struct {
	MaxConcurrency uint64
	FastCheck      bool
}

func snapshotCheckPath(snap *Snapshot, opts *CheckOptions, concurrency chan bool, wg *sync.WaitGroup) func(entrypath string, e *vfs.Entry, err error) error {
	return func(entrypath string, e *vfs.Entry, err error) error {
		concurrency <- true
		wg.Add(1)
		go func(entrypath string, e *vfs.Entry, err error) {
			defer func() { <-concurrency }()
			defer wg.Done()

			if err != nil {
				snap.Event(events.PathErrorEvent(snap.Header.Identifier, entrypath, err.Error()))
				return
			}

			snap.Event(events.PathEvent(snap.Header.Identifier, entrypath))

			if e.Stat().Mode().IsDir() {
				snap.Event(events.DirectoryEvent(snap.Header.Identifier, entrypath))
				snap.Event(events.DirectoryOKEvent(snap.Header.Identifier, entrypath))
				return
			}

			if !e.Stat().Mode().IsRegular() {
				return
			}

			snap.Event(events.FileEvent(snap.Header.Identifier, entrypath))
			concurrency <- true
			wg.Add(1)
			go func(_fileEntry *vfs.Entry, path string) {
				defer wg.Done()
				defer func() { <-concurrency }()

				object, err := snap.LookupObject(_fileEntry.Object)
				if err != nil {
					snap.Event(events.ObjectMissingEvent(snap.Header.Identifier, _fileEntry.Object))
					return
				}

				hasher := snap.repository.GetMACHasher()
				snap.Event(events.ObjectEvent(snap.Header.Identifier, object.ContentMAC))
				complete := true

				for _, chunk := range object.Chunks {
					snap.Event(events.ChunkEvent(snap.Header.Identifier, chunk.ContentMAC))
					if opts.FastCheck {
						if !snap.BlobExists(resources.RT_CHUNK, chunk.ContentMAC) {
							snap.Event(events.ChunkMissingEvent(snap.Header.Identifier, chunk.ContentMAC))
							complete = false
							break
						}
						snap.Event(events.ChunkOKEvent(snap.Header.Identifier, chunk.ContentMAC))
					} else {
						if !snap.BlobExists(resources.RT_CHUNK, chunk.ContentMAC) {
							snap.Event(events.ChunkMissingEvent(snap.Header.Identifier, chunk.ContentMAC))
							complete = false
							break
						}
						data, err := snap.GetBlob(resources.RT_CHUNK, chunk.ContentMAC)
						if err != nil {
							snap.Event(events.ChunkMissingEvent(snap.Header.Identifier, chunk.ContentMAC))
							complete = false
							break
						}
						snap.Event(events.ChunkOKEvent(snap.Header.Identifier, chunk.ContentMAC))

						hasher.Write(data)

						mac := snap.repository.ComputeMAC(data)
						if !bytes.Equal(mac[:], chunk.ContentMAC[:]) {
							snap.Event(events.ChunkCorruptedEvent(snap.Header.Identifier, chunk.ContentMAC))
							complete = false
							break
						}
					}
				}

				if !complete {
					snap.Event(events.ObjectCorruptedEvent(snap.Header.Identifier, object.ContentMAC))
				} else {
					snap.Event(events.ObjectOKEvent(snap.Header.Identifier, object.ContentMAC))
				}

				if !opts.FastCheck {
					if !bytes.Equal(hasher.Sum(nil), object.ContentMAC[:]) {
						snap.Event(events.ObjectCorruptedEvent(snap.Header.Identifier, object.ContentMAC))
						snap.Event(events.FileCorruptedEvent(snap.Header.Identifier, path))
						return
					}
				}
				snap.Event(events.FileOKEvent(snap.Header.Identifier, entrypath, e.Size()))
			}(e, entrypath)

		}(entrypath, e, err)
		return nil
	}
}

func (snap *Snapshot) Check(pathname string, opts *CheckOptions) (bool, error) {
	snap.Event(events.StartEvent())
	defer snap.Event(events.DoneEvent())

	fs, err := snap.Filesystem()
	if err != nil {
		return false, err
	}

	maxConcurrency := opts.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = uint64(snap.AppContext().MaxConcurrency)
	}

	maxConcurrencyChan := make(chan bool, maxConcurrency)
	wg := sync.WaitGroup{}
	defer wg.Wait()
	defer close(maxConcurrencyChan)

	err = fs.WalkDir(pathname, snapshotCheckPath(snap, opts, maxConcurrencyChan, &wg))
	if err != nil {
		return false, err
	}
	wg.Wait()

	return true, nil
}

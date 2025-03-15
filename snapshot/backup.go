package snapshot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/classifier"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository/state"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gobwas/glob"
)

type BackupContext struct {
	aborted        atomic.Bool
	abortedReason  error
	imp            importer.Importer
	maxConcurrency chan bool
	scanCache      *caching.ScanCache

	erridx   *btree.BTree[string, int, []byte]
	muerridx sync.Mutex

	xattridx   *btree.BTree[string, int, []byte]
	muxattridx sync.Mutex

	ctidx   *btree.BTree[string, int, objects.MAC]
	muctidx sync.Mutex
}

type BackupOptions struct {
	MaxConcurrency uint64
	Name           string
	Tags           []string
	Excludes       []glob.Glob
}

func (bc *BackupContext) recordEntry(entry *vfs.Entry) error {
	path := entry.Path()

	bytes, err := entry.ToBytes()
	if err != nil {
		return err
	}

	if entry.FileInfo.IsDir() {
		return bc.scanCache.PutDirectory(path, bytes)
	}
	return bc.scanCache.PutFile(path, bytes)
}

func (bc *BackupContext) recordError(path string, err error) error {
	entry := vfs.NewErrorItem(path, err.Error())
	serialized, e := entry.ToBytes()
	if e != nil {
		return err
	}

	bc.muerridx.Lock()
	e = bc.erridx.Insert(path, serialized)
	bc.muerridx.Unlock()
	return e
}

func (bc *BackupContext) recordXattr(record *importer.ScanRecord, objectMAC objects.MAC, size int64) error {
	xattr := vfs.NewXattr(record, objectMAC, size)
	serialized, err := xattr.ToBytes()
	if err != nil {
		return err
	}

	bc.muxattridx.Lock()
	err = bc.xattridx.Insert(xattr.ToPath(), serialized)
	bc.muxattridx.Unlock()
	return err
}

func (snapshot *Snapshot) skipExcludedPathname(options *BackupOptions, record *importer.ScanResult) bool {
	var pathname string
	switch {
	case record.Record != nil:
		pathname = record.Record.Pathname
	case record.Error != nil:
		pathname = record.Error.Pathname
	}

	if pathname == "/" {
		return false
	}

	doExclude := false
	for _, exclude := range options.Excludes {
		if exclude.Match(pathname) {
			doExclude = true
			break
		}
	}
	return doExclude
}

func (snap *Snapshot) importerJob(backupCtx *BackupContext, options *BackupOptions) (chan *importer.ScanRecord, error) {
	scanner, err := backupCtx.imp.Scan()
	if err != nil {
		return nil, err
	}

	workerCount := cap(backupCtx.maxConcurrency)

	filesChannel := make(chan *importer.ScanRecord, 1000)
	scannedResults := make(chan *importer.ScanResult, 1000)

	repoLocation := snap.repository.Location()
	var wg sync.WaitGroup

	// Worker pool to process scan results
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for record := range scannedResults {
				if backupCtx.aborted.Load() {
					continue
				}
				if snap.skipExcludedPathname(options, record) {
					continue
				}

				switch {
				case record.Error != nil:
					errRecord := record.Error
					if errRecord.Pathname == backupCtx.imp.Root() || len(errRecord.Pathname) < len(backupCtx.imp.Root()) {
						backupCtx.aborted.Store(true)
						backupCtx.abortedReason = errRecord.Err
						return
					}
					backupCtx.recordError(errRecord.Pathname, errRecord.Err)
					snap.Event(events.PathErrorEvent(snap.Header.Identifier, errRecord.Pathname, errRecord.Err.Error()))

				case record.Record != nil:
					fileRecord := record.Record
					snap.Event(events.PathEvent(snap.Header.Identifier, fileRecord.Pathname))

					if strings.HasPrefix(fileRecord.Pathname, repoLocation+"/") {
						snap.Logger().Warn("skipping entry from repository: %s", fileRecord.Pathname)
						continue
					}

					if !fileRecord.FileInfo.Mode().IsDir() {
						filesChannel <- fileRecord
					} else {
						entry := vfs.NewEntry(path.Dir(fileRecord.Pathname), fileRecord)
						if err := backupCtx.recordEntry(entry); err != nil {
							backupCtx.recordError(fileRecord.Pathname, err)
							return
						}
					}
				}
			}
		}()
	}

	// Scanner producer goroutine
	go func() {
		startEvent := events.StartImporterEvent()
		startEvent.SnapshotID = snap.Header.Identifier
		snap.Event(startEvent)

		var nFiles, nDirectories, size uint64
		for scanResult := range scanner {
			if backupCtx.aborted.Load() {
				break
			}
			scannedResults <- scanResult

			// Updating stats
			if scanResult.Record != nil {
				if scanResult.Record.FileInfo.Mode().IsRegular() {
					atomic.AddUint64(&nFiles, 1)
					atomic.AddUint64(&size, uint64(scanResult.Record.FileInfo.Size()))
				} else if scanResult.Record.FileInfo.IsDir() {
					atomic.AddUint64(&nDirectories, 1)
				}
			}
		}
		close(scannedResults)

		wg.Wait()           // Wait until workers have processed everything
		close(filesChannel) // Signal no more files will be sent

		doneEvent := events.DoneImporterEvent()
		doneEvent.SnapshotID = snap.Header.Identifier
		doneEvent.NumFiles = nFiles
		doneEvent.NumDirectories = nDirectories
		doneEvent.Size = size
		snap.Event(doneEvent)
	}()

	return filesChannel, nil
}

func (snap *Snapshot) importerJob2(backupCtx *BackupContext, options *BackupOptions) (chan *importer.ScanRecord, error) {
	scanner, err := backupCtx.imp.Scan()
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	filesChannel := make(chan *importer.ScanRecord, 1000)
	repoLocation := snap.repository.Location()

	go func() {
		startEvent := events.StartImporterEvent()
		startEvent.SnapshotID = snap.Header.Identifier
		snap.Event(startEvent)

		nFiles := uint64(0)
		nDirectories := uint64(0)
		size := uint64(0)
		for _record := range scanner {
			if backupCtx.aborted.Load() {
				break
			}
			if snap.skipExcludedPathname(options, _record) {
				continue
			}

			wg.Add(1)
			go func(record *importer.ScanResult) {
				defer func() {
					wg.Done()
				}()

				switch {
				case record.Error != nil:
					record := record.Error
					if record.Pathname == backupCtx.imp.Root() || len(record.Pathname) < len(backupCtx.imp.Root()) {
						backupCtx.aborted.Store(true)
						backupCtx.abortedReason = record.Err
						return
					}
					backupCtx.recordError(record.Pathname, record.Err)
					snap.Event(events.PathErrorEvent(snap.Header.Identifier, record.Pathname, record.Err.Error()))

				case record.Record != nil:
					record := record.Record
					snap.Event(events.PathEvent(snap.Header.Identifier, record.Pathname))

					if strings.HasPrefix(record.Pathname, repoLocation+"/") {
						snap.Logger().Warn("skipping entry from repository: %s", record.Pathname)
						// skip repository directory
						return
					}

					if !record.FileInfo.Mode().IsDir() {
						filesChannel <- record
						if !record.IsXattr {
							atomic.AddUint64(&nFiles, +1)
							if record.FileInfo.Mode().IsRegular() {
								atomic.AddUint64(&size, uint64(record.FileInfo.Size()))
							}
							// if snapshot root is a file, then reset to the parent directory
							if snap.Header.GetSource(0).Importer.Directory == record.Pathname {
								snap.Header.GetSource(0).Importer.Directory = filepath.Dir(record.Pathname)
							}
						}
					} else {
						atomic.AddUint64(&nDirectories, +1)
						entry := vfs.NewEntry(path.Dir(record.Pathname), record)
						if err := backupCtx.recordEntry(entry); err != nil {
							backupCtx.recordError(record.Pathname, err)
							return
						}
					}
				}
			}(_record)
		}
		wg.Wait()
		close(filesChannel)
		doneEvent := events.DoneImporterEvent()
		doneEvent.SnapshotID = snap.Header.Identifier
		doneEvent.NumFiles = nFiles
		doneEvent.NumDirectories = nDirectories
		doneEvent.Size = size
		snap.Event(doneEvent)
	}()

	return filesChannel, nil
}

func (snap *Snapshot) Backup(imp importer.Importer, options *BackupOptions) error {
	snap.Event(events.StartEvent())
	defer snap.Event(events.DoneEvent())

	done, err := snap.Lock()
	if err != nil {
		return err
	}
	defer snap.Unlock(done)

	vfsCache, err := snap.AppContext().GetCache().VFS(imp.Type(), imp.Origin())
	if err != nil {
		return err
	}

	cf, err := classifier.NewClassifier(snap.AppContext())
	if err != nil {
		return err
	}
	defer cf.Close()

	snap.Header.GetSource(0).Importer.Origin = imp.Origin()
	snap.Header.GetSource(0).Importer.Type = imp.Type()
	snap.Header.Tags = append(snap.Header.Tags, options.Tags...)

	if options.Name == "" {
		snap.Header.Name = imp.Root() + " @ " + snap.Header.GetSource(0).Importer.Origin
	} else {
		snap.Header.Name = options.Name
	}

	snap.Header.GetSource(0).Importer.Directory = imp.Root()

	maxConcurrency := options.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = uint64(snap.AppContext().MaxConcurrency)
	}

	backupCtx := &BackupContext{
		imp:            imp,
		maxConcurrency: make(chan bool, maxConcurrency),
		scanCache:      snap.scanCache,
	}

	errstore := caching.DBStore[string, []byte]{
		Prefix: "__error__",
		Cache:  snap.scanCache,
	}
	backupCtx.erridx, err = btree.New(&errstore, strings.Compare, 50)
	if err != nil {
		return err
	}

	xattrstore := caching.DBStore[string, []byte]{
		Prefix: "__xattr__",
		Cache:  snap.scanCache,
	}
	backupCtx.xattridx, err = btree.New(&xattrstore, vfs.PathCmp, 50)
	if err != nil {
		return err
	}

	ctstore := caching.DBStore[string, objects.MAC]{
		Prefix: "__contenttype__",
		Cache:  snap.scanCache,
	}
	backupCtx.ctidx, err = btree.New(&ctstore, strings.Compare, 50)
	if err != nil {
		return err
	}

	/* backup starts now */
	beginTime := time.Now()

	readJobs := make(chan ReadJob, 1000)
	processJobs := make(chan ProcessJob, 1000)
	uploadJobs := make(chan UploadJob, 2000)
	var readerWg, processorWg, uploaderWg sync.WaitGroup

	// Start readers
	for range maxConcurrency {
		readerWg.Add(1)
		go func() {
			defer readerWg.Done()
			snap.readerWorker(imp, readJobs, processJobs, backupCtx)
		}()
	}

	// Start processors
	for range maxConcurrency * 2 {
		processorWg.Add(1)
		go func() {
			defer processorWg.Done()
			snap.processWorker(processJobs, uploadJobs, backupCtx, cf, vfsCache)
		}()
	}

	// Start uploaders
	for range maxConcurrency * 4 {
		uploaderWg.Add(1)
		go func() {
			defer uploaderWg.Done()
			snap.uploaderWorker(uploadJobs)
		}()
	}

	/* importer */
	filesChannel, err := snap.importerJob(backupCtx, options)
	if err != nil {
		return err
	}

	// Feed readJobs
	for record := range filesChannel {
		readJobs <- ReadJob{record}
	}
	close(readJobs)

	// Close processJobs after readers
	go func() {
		readerWg.Wait()
		close(processJobs)
	}()

	// Close uploadJobs after processors
	go func() {
		processorWg.Wait()
		close(uploadJobs)
	}()

	// Wait for all pipeline stages to finish
	uploaderWg.Wait()

	errcsum, err := persistMACIndex(snap, backupCtx.erridx,
		resources.RT_ERROR_BTREE, resources.RT_ERROR_NODE, resources.RT_ERROR_ENTRY)
	if err != nil {
		return err
	}

	filestore := caching.DBStore[string, []byte]{
		Prefix: "__path__",
		Cache:  snap.scanCache,
	}
	fileidx, err := btree.New(&filestore, vfs.PathCmp, 50)
	if err != nil {
		return err
	}

	var rootSummary *vfs.Summary

	diriter := backupCtx.scanCache.EnumerateKeysWithPrefix("__directory__:", true)
	for dirPath, bytes := range diriter {
		select {
		case <-snap.AppContext().GetContext().Done():
			return snap.AppContext().GetContext().Err()
		default:
		}

		dirEntry, err := vfs.EntryFromBytes(bytes)
		if err != nil {
			return err
		}

		prefix := dirPath
		if prefix != "/" {
			prefix += "/"
		}

		childiter := backupCtx.scanCache.EnumerateKeysWithPrefix("__file__:"+prefix, false)

		for relpath, bytes := range childiter {
			if strings.Contains(relpath, "/") {
				continue
			}

			childEntry, err := vfs.EntryFromBytes(bytes)
			if err != nil {
				return err
			}

			childPath := prefix + relpath

			serialized, err := childEntry.ToBytes()
			if err != nil {
				return err
			}

			if err := fileidx.Insert(childPath, serialized); err != nil && err != btree.ErrExists {
				return err
			}

			data, err := vfsCache.GetFileSummary(childPath)
			if err != nil {
				continue
			}

			fileSummary, err := vfs.FileSummaryFromBytes(data)
			if err != nil {
				continue
			}

			dirEntry.Summary.Directory.Children++
			dirEntry.Summary.UpdateWithFileSummary(fileSummary)
		}

		subDirIter := backupCtx.scanCache.EnumerateKeysWithPrefix("__directory__:"+prefix, false)
		for relpath := range subDirIter {
			if relpath == "" || strings.Contains(relpath, "/") {
				continue
			}

			childPath := prefix + relpath
			data, err := snap.scanCache.GetSummary(childPath)
			if err != nil {
				continue
			}

			childSummary, err := vfs.SummaryFromBytes(data)
			if err != nil {
				continue
			}
			dirEntry.Summary.Directory.Children++
			dirEntry.Summary.Directory.Directories++
			dirEntry.Summary.UpdateBelow(childSummary)
		}

		erriter, err := backupCtx.erridx.ScanFrom(prefix)
		if err != nil {
			return err
		}
		for erriter.Next() {
			path, _ := erriter.Current()
			if !strings.HasPrefix(path, prefix) {
				break
			}
			if strings.Contains(path[len(prefix):], "/") {
				break
			}
			dirEntry.Summary.Below.Errors++
		}
		if err := erriter.Err(); err != nil {
			return err
		}

		dirEntry.Summary.UpdateAverages()

		classifications := cf.Processor(dirPath).Directory(dirEntry)
		for _, result := range classifications {
			dirEntry.AddClassification(result.Analyzer, result.Classes)
		}

		serializedSummary, err := dirEntry.Summary.ToBytes()
		if err != nil {
			backupCtx.recordError(dirPath, err)
			return err
		}

		err = snap.scanCache.PutSummary(dirPath, serializedSummary)
		if err != nil {
			backupCtx.recordError(dirPath, err)
			return err
		}

		snap.Event(events.DirectoryOKEvent(snap.Header.Identifier, dirPath))
		if dirPath == "/" {
			if rootSummary != nil {
				panic("double /!")
			}
			rootSummary = dirEntry.Summary
		}

		serialized, err := dirEntry.ToBytes()
		if err != nil {
			return err
		}

		mac := snap.repository.ComputeMAC(serialized)
		if err := snap.PutBlobIfNotExists(resources.RT_VFS_ENTRY, mac, serialized); err != nil {
			return err
		}

		if err := fileidx.Insert(dirPath, serialized); err != nil && err != btree.ErrExists {
			return err
		}

		if err := backupCtx.recordEntry(dirEntry); err != nil {
			return err
		}
	}

	rootcsum, err := persistIndex(snap, fileidx, resources.RT_VFS_BTREE,
		resources.RT_VFS_NODE, func(data []byte) (objects.MAC, error) {
			return snap.repository.ComputeMAC(data), nil
		})
	if err != nil {
		return err
	}

	xattrcsum, err := persistMACIndex(snap, backupCtx.xattridx,
		resources.RT_XATTR_BTREE, resources.RT_XATTR_NODE, resources.RT_XATTR_ENTRY)
	if err != nil {
		return err
	}

	ctmac, err := persistIndex(snap, backupCtx.ctidx, resources.RT_BTREE_ROOT, resources.RT_BTREE_NODE, func(mac objects.MAC) (objects.MAC, error) {
		return mac, nil
	})
	if err != nil {
		return err
	}

	if backupCtx.aborted.Load() {
		return backupCtx.abortedReason
	}

	snap.Header.GetSource(0).VFS = header.VFS{
		Root:   rootcsum,
		Xattrs: xattrcsum,
		Errors: errcsum,
	}
	snap.Header.Duration = time.Since(beginTime)
	snap.Header.GetSource(0).Summary = *rootSummary
	snap.Header.GetSource(0).Indexes = []header.Index{
		{
			Name:  "content-type",
			Type:  "btree",
			Value: ctmac,
		},
	}

	return snap.Commit()
}

func entropy(data []byte) (float64, [256]float64) {
	if len(data) == 0 {
		return 0.0, [256]float64{}
	}

	// Count the frequency of each byte value
	var freq [256]float64
	for _, b := range data {
		freq[b]++
	}

	// Calculate the entropy
	entropy := 0.0
	dataSize := float64(len(data))
	for _, f := range freq {
		if f > 0 {
			p := f / dataSize
			entropy -= p * math.Log2(p)
		}
	}
	return entropy, freq
}

func (snap *Snapshot) chunkify(rd io.ReadCloser, record *importer.ScanRecord, uploadJobs chan<- UploadJob) (*objects.Object, []byte, objects.MAC, error) {
	defer rd.Close()

	object := objects.NewObject()
	object.ContentType = mime.TypeByExtension(path.Ext(record.Pathname))

	objectHasher := snap.repository.GetMACHasher()

	var firstChunk = true
	var totalEntropy float64
	var totalDataSize uint64

	processChunk := func(data []byte) error {
		chunkMAC := snap.repository.ComputeMAC(data)

		if firstChunk {
			if object.ContentType == "" {
				object.ContentType = mimetype.Detect(data).String()
			}
			firstChunk = false
		}

		objectHasher.Write(data)

		entropyScore, _ := entropy(data)
		totalEntropy += entropyScore * float64(len(data))
		totalDataSize += uint64(len(data))

		chunk := objects.NewChunk()
		chunk.ContentMAC = chunkMAC
		chunk.Length = uint32(len(data))
		chunk.Entropy = entropyScore
		object.Chunks = append(object.Chunks, *chunk)

		// Immediately enqueue chunk for upload
		uploadJobs <- UploadJob{
			resourceType: resources.RT_CHUNK,
			mac:          chunkMAC,
			data:         data,
		}

		return nil
	}

	if record.FileInfo.Size() == 0 {
		if err := processChunk([]byte{}); err != nil {
			return nil, nil, objects.MAC{}, err
		}
	} else if record.FileInfo.Size() < int64(snap.repository.Configuration().Chunking.MinSize) {
		buf, err := io.ReadAll(rd)
		if err != nil {
			return nil, nil, objects.MAC{}, err
		}
		if err := processChunk(buf); err != nil {
			return nil, nil, objects.MAC{}, err
		}
	} else {
		chk, err := snap.repository.Chunker(rd)
		if err != nil {
			return nil, nil, objects.MAC{}, err
		}

		for {
			chunkData, err := chk.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, nil, objects.MAC{}, err
			}
			if err := processChunk(chunkData); err != nil {
				return nil, nil, objects.MAC{}, err
			}
		}
	}

	if totalDataSize > 0 {
		object.Entropy = totalEntropy / float64(totalDataSize)
	} else {
		object.Entropy = 0.0
	}

	objectMAC := snap.repository.ComputeMAC(objectHasher.Sum(nil))
	object.ContentMAC = objectMAC

	serializedObject, err := object.Serialize()
	if err != nil {
		return nil, nil, objects.MAC{}, err
	}

	objectSerializedMAC := snap.repository.ComputeMAC(serializedObject)

	// Upload serialized object metadata immediately
	uploadJobs <- UploadJob{
		resourceType: resources.RT_OBJECT,
		mac:          objectSerializedMAC,
		data:         serializedObject,
	}

	return object, serializedObject, objectSerializedMAC, nil
}

func (snap *Snapshot) PutPackfile(packer *Packer) error {

	repo := snap.repository

	serializedData, err := packer.Packfile.SerializeData()
	if err != nil {
		return fmt.Errorf("could not serialize pack file data %s", err.Error())
	}
	serializedIndex, err := packer.Packfile.SerializeIndex()
	if err != nil {
		return fmt.Errorf("could not serialize pack file index %s", err.Error())
	}
	serializedFooter, err := packer.Packfile.SerializeFooter()
	if err != nil {
		return fmt.Errorf("could not serialize pack file footer %s", err.Error())
	}

	encryptedIndex, err := repo.EncodeBuffer(serializedIndex)
	if err != nil {
		return err
	}

	encryptedFooter, err := repo.EncodeBuffer(serializedFooter)
	if err != nil {
		return err
	}

	serializedPackfile := append(serializedData, encryptedIndex...)
	serializedPackfile = append(serializedPackfile, encryptedFooter...)

	/* it is necessary to track the footer _encrypted_ length */
	encryptedFooterLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(encryptedFooterLength, uint32(len(encryptedFooter)))
	serializedPackfile = append(serializedPackfile, encryptedFooterLength...)

	mac := snap.repository.ComputeMAC(serializedPackfile)

	repo.Logger().Trace("snapshot", "%x: PutPackfile(%x, ...)", snap.Header.GetIndexShortID(), mac)
	err = snap.repository.PutPackfile(mac, bytes.NewBuffer(serializedPackfile))
	if err != nil {
		return fmt.Errorf("could not write pack file %s", err.Error())
	}

	for _, Type := range packer.Types() {
		for blobMAC := range packer.Blobs[Type] {
			for idx, blob := range packer.Packfile.Index {
				if blob.MAC == blobMAC && blob.Type == Type {
					delta := state.DeltaEntry{
						Type:    blob.Type,
						Version: packer.Packfile.Index[idx].Version,
						Blob:    blobMAC,
						Location: state.Location{
							Packfile: mac,
							Offset:   packer.Packfile.Index[idx].Offset,
							Length:   packer.Packfile.Index[idx].Length,
						},
					}

					if err := snap.deltaState.PutDelta(delta); err != nil {
						return err
					}

					break
				}
			}
		}
	}

	if err := snap.deltaState.PutPackfile(snap.Header.Identifier, mac); err != nil {
		return err
	}

	return nil
}

func (snap *Snapshot) Commit() error {
	repo := snap.repository

	serializedHdr, err := snap.Header.Serialize()
	if err != nil {
		return err
	}

	if kp := snap.AppContext().Keypair; kp != nil {
		serializedHdrMAC := snap.repository.ComputeMAC(serializedHdr)
		signature := kp.Sign(serializedHdrMAC[:])
		if err := snap.PutBlob(resources.RT_SIGNATURE, snap.Header.Identifier, signature); err != nil {
			return err
		}
	}

	if err := snap.PutBlob(resources.RT_SNAPSHOT, snap.Header.Identifier, serializedHdr); err != nil {
		return err
	}
	snap.packerManager.Wait()

	stateDelta := snap.buildSerializedDeltaState()
	err = repo.PutState(snap.Header.Identifier, stateDelta)
	if err != nil {
		snap.Logger().Warn("Failed to push the state to the repository %s", err)
		return err
	}

	snap.Logger().Trace("snapshot", "%x: Commit()", snap.Header.GetIndexShortID())
	return nil
}

func (snap *Snapshot) buildSerializedDeltaState() io.Reader {
	pr, pw := io.Pipe()

	/* By using a pipe and a goroutine we bound the max size in memory. */
	go func() {
		defer pw.Close()
		if err := snap.deltaState.SerializeToStream(pw); err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr
}

type ReadJob struct {
	record *importer.ScanRecord
}

type ProcessJob struct {
	record *importer.ScanRecord
	data   io.ReadCloser
}

type UploadJob struct {
	resourceType resources.Type
	mac          objects.MAC
	data         []byte
}

func (snap *Snapshot) readerWorker(imp importer.Importer, readJobs <-chan ReadJob, processJobs chan<- ProcessJob, backupCtx *BackupContext) {
	for job := range readJobs {
		select {
		case <-snap.AppContext().GetContext().Done():
			return
		default:
		}

		reader, err := imp.NewReader(job.record.Pathname)
		if err != nil {
			backupCtx.recordError(job.record.Pathname, err)
			continue
		}
		processJobs <- ProcessJob{record: job.record, data: reader}
	}
}

func (snap *Snapshot) processWorker(processJobs <-chan ProcessJob, uploadJobs chan<- UploadJob, backupCtx *BackupContext, cf *classifier.Classifier, vfsCache *caching.VFSCache) {
	for job := range processJobs {
		func() {
			defer job.data.Close()
			record := job.record
			pathname := record.Pathname

			snap.Event(events.FileEvent(snap.Header.Identifier, pathname))

			var err error
			var fileEntry *vfs.Entry
			var object *objects.Object
			var objectMAC objects.MAC
			var objectSerialized []byte

			var cachedFileEntry *vfs.Entry
			var cachedFileEntryMAC objects.MAC

			// Check if the file entry and underlying objects are already in the cache
			if data, err := vfsCache.GetFilename(record.Pathname); err != nil {
				snap.Logger().Warn("VFS CACHE: Error getting filename: %v", err)
			} else if data != nil {
				cachedFileEntry, err = vfs.EntryFromBytes(data)
				if err != nil {
					snap.Logger().Warn("VFS CACHE: Error unmarshaling filename: %v", err)
				} else {
					cachedFileEntryMAC = snap.repository.ComputeMAC(data)
					if cachedFileEntry.Stat().Equal(&record.FileInfo) {
						fileEntry = cachedFileEntry
						if fileEntry.FileInfo.Mode().IsRegular() {
							data, err := vfsCache.GetObject(cachedFileEntry.Object)
							if err != nil {
								snap.Logger().Warn("VFS CACHE: Error getting object: %v", err)
							} else if data != nil {
								cachedObject, err := objects.NewObjectFromBytes(data)
								if err != nil {
									snap.Logger().Warn("VFS CACHE: Error unmarshaling object: %v", err)
								} else {
									object = cachedObject
									objectMAC = snap.Repository().ComputeMAC(data)
									objectSerialized = data
								}
							}
						}
					}
				}
			}

			if object != nil {
				err := snap.PutBlobIfNotExists(resources.RT_OBJECT, objectMAC, objectSerialized)
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}
			}

			if record.FileInfo.Mode().IsRegular() {
				if object == nil || !snap.BlobExists(resources.RT_OBJECT, objectMAC) {
					object, objectSerialized, objectMAC, err = snap.chunkify(job.data, job.record, uploadJobs)
					if err != nil {
						backupCtx.recordError(record.Pathname, err)
						return
					}
					if err := vfsCache.PutObject(objectMAC, objectSerialized); err != nil {
						backupCtx.recordError(record.Pathname, err)
						return
					}

					uploadJobs <- UploadJob{resources.RT_OBJECT, objectMAC, objectSerialized}
				}
			}

			// Handle special cases (xattrs)
			if record.IsXattr {
				backupCtx.recordXattr(record, objectMAC, object.Size())
				return
			}

			// VFS entry handling
			if fileEntry == nil || !snap.BlobExists(resources.RT_VFS_ENTRY, cachedFileEntryMAC) {
				fileEntry = vfs.NewEntry(path.Dir(pathname), record)
				if object != nil {
					fileEntry.Object = objectMAC
				}

				classifications := cf.Processor(pathname).File(fileEntry)
				for _, result := range classifications {
					fileEntry.AddClassification(result.Analyzer, result.Classes)
				}

				serialized, err := fileEntry.ToBytes()
				if err != nil {
					backupCtx.recordError(pathname, err)
					return
				}

				fileEntryMAC := snap.repository.ComputeMAC(serialized)

				uploadJobs <- UploadJob{resources.RT_VFS_ENTRY, fileEntryMAC, serialized}

				err = vfsCache.PutFilename(record.Pathname, serialized)
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}

				fileSummary := &vfs.FileSummary{
					Size:    uint64(record.FileInfo.Size()),
					Mode:    record.FileInfo.Mode(),
					ModTime: record.FileInfo.ModTime().Unix(),
				}
				if object != nil {
					fileSummary.Objects++
					fileSummary.Chunks += uint64(len(object.Chunks))
					fileSummary.ContentType = object.ContentType
					fileSummary.Entropy = object.Entropy
				}

				seralizedFileSummary, err := fileSummary.Serialize()
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}

				err = vfsCache.PutFileSummary(record.Pathname, seralizedFileSummary)
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}
			}

			if object != nil {
				parts := strings.SplitN(object.ContentType, ";", 2)
				mime := parts[0]

				k := fmt.Sprintf("/%s%s", mime, fileEntry.Path())
				bytes, err := fileEntry.ToBytes()
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}
				backupCtx.muctidx.Lock()
				err = backupCtx.ctidx.Insert(k, snap.repository.ComputeMAC(bytes))
				backupCtx.muctidx.Unlock()
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}
			}

			if err := backupCtx.recordEntry(fileEntry); err != nil {
				backupCtx.recordError(record.Pathname, err)
				return
			}

			snap.Event(events.FileOKEvent(snap.Header.Identifier, record.Pathname, record.FileInfo.Size()))
		}()
	}
}

func (snap *Snapshot) uploaderWorker(uploadJobs <-chan UploadJob) {
	for job := range uploadJobs {
		if err := snap.PutBlobIfNotExists(job.resourceType, job.mac, job.data); err != nil {
			snap.Logger().Warn("Upload failed (%x): %v", job.mac, err)
		}
	}
}

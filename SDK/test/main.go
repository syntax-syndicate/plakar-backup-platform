package main

import (
	"bytes"
	"fmt"
	"github.com/PlakarKorp/plakar/objects"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/PlakarKorp/plakar/SDK"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/pkg/xattr"
)

type TestFSImporter struct {
	ctx     *appcontext.AppContext
	rootDir string

	uidToName map[uint64]string
	gidToName map[uint64]string
	mu        sync.RWMutex
}

func (p *TestFSImporter) Origin() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	return hostname
}

func (p *TestFSImporter) Type() string {
	return "fs"
}

func (p *TestFSImporter) Scan() (any, error) {
	fmt.Println("Scan called")
	realp, err := p.realpathFollow(p.rootDir)
	if err != nil {
		return nil, err
	}

	results := make(chan *importer.ScanResult, 1000)
	go p.walkDir_walker(results, p.rootDir, realp, 256)
	println("\n\n")

	res := <-results
	if res.Record != nil {
		return *res.Record, nil
	}
	return res.Error, nil
}

func (f *TestFSImporter) walkDir_walker(results chan<- *importer.ScanResult, rootDir, realp string, numWorkers int) {
	jobs := make(chan string, 1000) // Buffered channel to feed paths to workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go f.walkDir_worker(jobs, results, &wg)
	}

	// Add prefix directories first
	walkDir_addPrefixDirectories(realp, jobs, results)
	if realp != rootDir {
		jobs <- rootDir
		walkDir_addPrefixDirectories(rootDir, jobs, results)
	}

	err := filepath.WalkDir(realp, func(path string, d fs.DirEntry, err error) error {
		if f.ctx.Err() != nil {
			return err
		}

		if err != nil {
			results <- importer.NewScanError(path, err)
			return nil
		}
		jobs <- path
		return nil
	})
	if err != nil {
		results <- importer.NewScanError(realp, err)
	}

	close(jobs)
	wg.Wait()
	close(results)
}

func (p *TestFSImporter) lookupIDs(uid, gid uint64) (uname, gname string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if name, ok := p.uidToName[uid]; !ok {
		if u, err := user.LookupId(fmt.Sprint(uid)); err == nil {
			uname = u.Username

			p.mu.RUnlock()
			p.mu.Lock()
			p.uidToName[uid] = uname
			p.mu.Unlock()
			p.mu.RLock()
		}
	} else {
		uname = name
	}

	if name, ok := p.gidToName[gid]; !ok {
		if g, err := user.LookupGroupId(fmt.Sprint(gid)); err == nil {
			gname = g.Name

			p.mu.RUnlock()
			p.mu.Lock()
			p.gidToName[gid] = name
			p.mu.Unlock()
			p.mu.RLock()
		}
	} else {
		gname = name
	}

	return
}

func (f *TestFSImporter) realpathFollow(path string) (resolved string, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		realpath, err := os.Readlink(path)
		if err != nil {
			return "", err
		}

		if !filepath.IsAbs(realpath) {
			realpath = filepath.Join(filepath.Dir(path), realpath)
		}
		path = realpath
	}

	return path, nil
}

func (p *TestFSImporter) NewReader(pathname string) (io.ReadCloser, error) {
	fmt.Println("NewReader called with pathname:", pathname)
	if pathname[0] == '/' && runtime.GOOS == "windows" {
		pathname = pathname[1:]
	}
	return os.Open(pathname)
}

func (p *TestFSImporter) NewExtendedAttributeReader(pathname string, attribute string) (io.ReadCloser, error) {
	if pathname[0] == '/' && runtime.GOOS == "windows" {
		pathname = pathname[1:]
	}

	data, err := xattr.Get(pathname, attribute)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (p *TestFSImporter) Close() error {
	return nil
}

func (p *TestFSImporter) Root() string {
	return p.rootDir
}

func (f *TestFSImporter) walkDir_worker(jobs <-chan string, results chan<- *importer.ScanResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		var (
			path string
			ok   bool
		)

		select {
		case path, ok = <-jobs:
			if !ok {
				return
			}
		case <-f.ctx.Done():
			return
		}

		info, err := os.Lstat(path)
		if err != nil {
			results <- importer.NewScanError(path, err)
			continue
		}

		extendedAttributes, err := xattr.List(path)
		if err != nil {
			results <- importer.NewScanError(path, err)
			continue
		}

		fileinfo := objects.FileInfoFromStat(info)
		fileinfo.Lusername, fileinfo.Lgroupname = f.lookupIDs(fileinfo.Uid(), fileinfo.Gid())

		var originFile string
		if fileinfo.Mode()&os.ModeSymlink != 0 {
			originFile, err = os.Readlink(path)
			if err != nil {
				results <- importer.NewScanError(path, err)
				continue
			}
		}
		results <- importer.NewScanRecord(filepath.ToSlash(path), originFile, fileinfo, extendedAttributes)
		for _, attr := range extendedAttributes {
			results <- importer.NewScanXattr(filepath.ToSlash(path), attr, objects.AttributeExtended)
		}
	}
}

func walkDir_addPrefixDirectories(rootDir string, jobs chan<- string, results chan<- *importer.ScanResult) {
	atoms := strings.Split(rootDir, string(os.PathSeparator))

	for i := range len(atoms) - 1 {
		path := filepath.Join(atoms[0 : i+1]...)

		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		if _, err := os.Stat(path); err != nil {
			results <- importer.NewScanError(path, err)
			continue
		}

		jobs <- path
	}
}

func main() {
	fmt.Println("Hello, Plakar SDK! 1")
	fsImporter := TestFSImporter{
		ctx:       appcontext.NewAppContext(),
		rootDir:   "/tmp",
		uidToName: make(map[uint64]string),
		gidToName: make(map[uint64]string),
	}
	fmt.Println("Hello, Plakar SDK! 2")
	server := &sdk.Server{
		Sdk: &sdk.PlakarImporterSDK{}, // <-- Properly initialize Sdk
	}
	fmt.Println("Hello, Plakar SDK! 3")
	server.Sdk.SetScan(fsImporter.Scan)
	fmt.Println("Hello, Plakar SDK! 4")
	server.Sdk.SetNewReader(fsImporter.NewReader)
	fmt.Println("Hello, Plakar SDK! 5")
	server.Run()
}

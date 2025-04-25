package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/connectors/data/fs"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/cockroachdb/pebble/v2"
	"github.com/vmihailenco/msgpack/v5"
)

type empty struct{}

type Node = btree.Node[string, int, empty]

type pebbleStore struct {
	counter int
	db      *pebble.DB
}

func (l *pebbleStore) Get(i int) (*Node, error) {
	key := fmt.Sprintf("%d", i)
	bytes, closer, err := l.db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	node := &Node{}
	err = msgpack.Unmarshal(bytes, node)
	closer.Close()
	return node, err
}

func (l *pebbleStore) Update(i int, node *Node) error {
	key := fmt.Sprintf("%d", i)
	bytes, err := msgpack.Marshal(node)
	if err != nil {
		return err
	}
	return l.db.Set([]byte(key), bytes, nil)
}

func (l *pebbleStore) Put(node *Node) (int, error) {
	n := l.counter
	key := fmt.Sprintf("%d", n)
	l.counter++

	bytes, err := msgpack.Marshal(node)
	if err != nil {
		return 0, err
	}
	return n, l.db.Set([]byte(key), bytes, nil)
}

func main() {
	var (
		verify  bool
		xattr   bool
		dbpath  string
		order   int
		dot     string
		memprof string
		cpuprof string
	)
	flag.BoolVar(&verify, "verify", false, `Whether to verify the tree at the end`)
	flag.BoolVar(&xattr, "xattr", false, `get xattr for all the files as well`)
	flag.StringVar(&dbpath, "dbpath", "/tmp/pebble", `Path to the pebble db directory; use "memory" for an in-memory btree`)
	flag.IntVar(&order, "order", 50, `Order of the btree`)
	flag.StringVar(&dot, "dot", "", `where to put the generated dot; empty for none`)
	flag.StringVar(&cpuprof, "profile-cpu", "", "profile CPU usage")
	flag.StringVar(&memprof, "profile-mem", "", "profile MEM usage")
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("Missig directory to import")
	}

	if cpuprof != "" {
		f, err := os.Create(cpuprof)
		if err != nil {
			log.Fatalf("%s: could not create CPU profile: %s\n", flag.CommandLine.Name(), err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("%s: could not start CPU profile: %s\n", flag.CommandLine.Name(), err)
		}
		defer pprof.StopCPUProfile()
	}

	if memprof != "" {
		f, err := os.Create(memprof)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatalf("%s: could not write MEM profile: %d\n", flag.CommandLine.Name(), err)
		}
	}

	var store btree.Storer[string, int, empty]
	if dbpath == "memory" {
		store = &btree.InMemoryStore[string, empty]{}
	} else {
		os.Remove(dbpath)
		if err := os.MkdirAll(dbpath, 0755); err != nil {
			log.Fatalf("can't mkdirall %s: %s", dbpath, err)
		}
		db, err := pebble.Open(dbpath, nil)
		if err != nil {
			log.Fatal("failed to open the pebble:", err)
		}
		store = &pebbleStore{db: db}
		defer os.Remove(dbpath)
	}

	idx, err := btree.New(store, vfs.PathCmp, order)
	if err != nil {
		log.Fatal("Failed to create the btree:", err)
	}

	imp, err := fs.NewFSImporter(appcontext.NewAppContext(), "fs", map[string]string{"location": flag.Arg(0)})
	if err != nil {
		log.Fatal("new fs importer failed:", err)
	}

	scan, err := imp.Scan()
	if err != nil {
		log.Fatal("fs scan failed:", err)
	}

	var items uint64
	log.Println("starting the scan")
	for record := range scan {
		switch {
		case record.Error != nil:
			log.Print("failed to scan:", record.Error.Pathname)
			continue
		case record.Record != nil:
			path := record.Record.Pathname
			if err := idx.Insert(path, empty{}); err != nil && err != btree.ErrExists {
				log.Fatalf("failed to insert %s: %v", path, err)
			}
			items++

			if xattr && record.Record.IsXattr {
				rd, err := imp.NewExtendedAttributeReader(path, record.Record.XattrName)
				if err != nil {
					log.Fatalln("failed to get xattr for", path, "due to", err)
				}
				rd.Close()
				log.Println(path, "found xattr named", record.Record.XattrName)
			}
		default:
			log.Fatalln("got unknown scanrecord", record)
		}
	}
	log.Println("scan finished.", items, "items scanned")

	if dot != "" {
		fp, err := os.Create(dot)
		if err != nil {
			log.Printf("failed to open %s: %v", dot, err)
		}
		fmt.Fprintln(fp, "digraph g {")
		idx.Dot(fp, false)
		fmt.Fprintln(fp, "}")
	}

	if verify {
		if err := idx.Verify(); err != nil {
			log.Fatalln("verify failed:", err)
		}
	}
}

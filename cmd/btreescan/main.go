package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/snapshot/importer/fs"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vmihailenco/msgpack/v5"
)

type empty struct{}

type Node = btree.Node[string, int, empty]

type leveldbstore struct {
	counter int
	db      *leveldb.DB
}

func (l *leveldbstore) Get(i int) (*Node, error) {
	key := fmt.Sprintf("%d", i)
	bytes, err := l.db.Get([]byte(key), nil)
	if err != nil {
		return nil, err
	}
	node := &Node{}
	err = msgpack.Unmarshal(bytes, node)
	return node, err
}

func (l *leveldbstore) Update(i int, node *Node) error {
	key := fmt.Sprintf("%d", i)
	bytes, err := msgpack.Marshal(node)
	if err != nil {
		return err
	}
	return l.db.Put([]byte(key), bytes, nil)
}

func (l *leveldbstore) Put(node *Node) (int, error) {
	n := l.counter
	key := fmt.Sprintf("%d", n)
	l.counter++

	bytes, err := msgpack.Marshal(node)
	if err != nil {
		return 0, err
	}
	return n, l.db.Put([]byte(key), bytes, nil)
}

func main() {
	var (
		verify bool
		dbpath string
		order  int
		dot    string
	)
	flag.BoolVar(&verify, "verify", false, `Whether to verify the tree at the end`)
	flag.StringVar(&dbpath, "dbpath", "/tmp/leveldb", `Path to the leveldb; use "memory" for an in-memory btree`)
	flag.IntVar(&order, "order", 50, `Order of the btree`)
	flag.StringVar(&dot, "dot", "", `where to put the generated dot; empty for none`)
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("Missig directory to import")
	}

	var store btree.Storer[string, int, empty]
	if dbpath == "memory" {
		store = &btree.InMemoryStore[string, empty]{}
	} else {
		os.Remove(dbpath)
		db, err := leveldb.OpenFile(dbpath, nil)
		if err != nil {
			log.Fatal("failed to open the leveldb:", err)
		}
		store = &leveldbstore{db: db}
		defer os.Remove(dbpath)
	}

	idx, err := btree.New(store, vfs.PathCmp, order)
	if err != nil {
		log.Fatal("Failed to create the btree:", err)
	}

	imp, err := fs.NewFSImporter(flag.Arg(0))
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
		switch record := record.(type) {
		case importer.ScanError:
			log.Print("failed to scan:", record.Pathname)
			continue
		case importer.ScanRecord:
			path := record.Pathname
			if err := idx.Insert(path, empty{}); err != nil && err != btree.ErrExists {
				log.Fatalf("failed to insert %s: %v", path, err)
			}
			items++
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

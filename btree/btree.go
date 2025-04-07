package btree

import (
	"fmt"
	"io"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/vmihailenco/msgpack/v5"
)

const BTREE_VERSION = "1.0.0"
const NODE_VERSION = "1.0.0"

func init() {
	versioning.Register(resources.RT_BTREE_ROOT, versioning.FromString(BTREE_VERSION))
	versioning.Register(resources.RT_BTREE_NODE, versioning.FromString(NODE_VERSION))
}

type Storer[K any, P comparable, V any] interface {
	// Get returns the node pointed by P.  The pointer is one
	// previously returned by the Put method.
	Get(P) (*Node[K, P, V], error)
	// Updates in-place the node pointed by P.
	Update(P, *Node[K, P, V]) error
	// Put saves a new node and returns its address, or an error.
	Put(*Node[K, P, V]) (P, error)
}

type OpCode int

const (
	OpAdd OpCode = iota
)

type Op[K, V any] struct {
	Opcode OpCode
	Key    K
	Val    V
}

type Node[K any, P comparable, V any] struct {
	Version versioning.Version `msgpack:"version"`

	// An intermediate node has only Keys and Pointers, while
	// leaves have only keys and values and optionally a next
	// pointer.
	//
	// invariant: len(Pointers) == len(Keys) + 1 in intermediate nodes
	// invariant: len(Values)   == len(Keys)     in leaf nodes
	Keys     []K `msgpack:"keys"`
	Pointers []P `msgpack:"pointers"`
	Values   []V `msgpack:"values"`
	Prev     *P  `msgpack:"prev,omitempty"` // unused for now
	Next     *P  `msgpack:"next,omitempty"`

	// invariant: len(Ops) < order
	Ops []Op[K, V]
}

// BTree implements a B+tree.  K is the type for the key, V for the
// value stored, and P is a pointer type: it could be a disk sector,
// a MAC in a packfile, or a key in a leveldb cache.  or more.
type BTree[K any, P comparable, V any] struct {
	Version versioning.Version
	Order   int
	Count   int
	Root    P
	cache   *cache[K, P, V]
	compare func(K, K) int
	mtx     sync.Mutex
}

// New returns a new, empty tree.
func New[K any, P comparable, V any](store Storer[K, P, V], compare func(K, K) int, order int) (*BTree[K, P, V], error) {
	root := Node[K, P, V]{
		Version: versioning.FromString(NODE_VERSION),
	}
	ptr, err := store.Put(&root)
	if err != nil {
		return nil, err
	}

	return &BTree[K, P, V]{
		Order:   order,
		Root:    ptr,
		cache:   cachefor(store, order),
		compare: compare,
	}, nil
}

// FromStorage returns a btree from the given storage.  The root must
// exist, eventually empty, i.e. it should be a tree previously
// created via New().
func FromStorage[K any, P comparable, V any](root P, store Storer[K, P, V], compare func(K, K) int, order int) *BTree[K, P, V] {
	return &BTree[K, P, V]{
		Version: versioning.FromString(BTREE_VERSION),
		Order:   order,
		Root:    root,
		cache:   cachefor(store, order),
		compare: compare,
	}
}

func Deserialize[K any, P comparable, V any](rd io.Reader, store Storer[K, P, V], compare func(K, K) int) (*BTree[K, P, V], error) {
	var root BTree[K, P, V]
	if err := msgpack.NewDecoder(rd).Decode(&root); err != nil {
		return nil, err
	}
	return FromStorage(root.Root, store, compare, root.Order), nil
}

func (b *BTree[K, P, V]) Close() error {
	return b.cache.Close()
}

func newNodeFrom[K any, P comparable, V any](keys []K, pointers []P, values []V) *Node[K, P, V] {
	node := &Node[K, P, V]{
		Version:  versioning.FromString(NODE_VERSION),
		Keys:     make([]K, len(keys)),
		Pointers: make([]P, len(pointers)),
		Values:   make([]V, len(values)),
	}
	copy(node.Keys, keys)
	copy(node.Pointers, pointers)
	copy(node.Values, values)
	return node
}

func (n *Node[K, P, V]) isleaf() bool {
	return len(n.Pointers) == 0
}

func (b *BTree[K, P, V]) pickforflush(node *Node[K, P, V]) (dest P, ops []Op[K, V]) {
	var (
		tot    = make([]int, len(node.Pointers))
		dests  = make([]int, len(node.Ops))
		newops = make([]Op[K, V], 0, len(node.Ops))
		max    int
		target int
	)

	if node.isleaf() {
		panic("pickforflush on a leaf!")
	}

	if len(node.Keys) + 1 != len(node.Pointers) {
		panic(fmt.Sprintf("invariant broken (keys %d; pointers %d; values %d)",
			len(node.Keys), len(node.Pointers), len(node.Values)))
	}

	for i := range node.Ops {
		idx, found := slices.BinarySearchFunc(node.Keys, node.Ops[i].Key, b.compare)
		if found {
			idx++
		}
		dests[i] = idx
		if tot[idx]++; tot[idx] > max {
			max = tot[idx]
			target = idx
		}
	}

	for i := 0; i < len(node.Ops); i++ {
		if dests[i] == target && len(ops) < b.Order {
			ops = append(ops, node.Ops[i])
		} else {
			newops = append(newops, node.Ops[i])
		}
	}
	log.Printf("got %d ops; now we have %d newop and %d flushed ops", len(node.Ops), len(newops), len(ops))

	node.Ops = newops
	return node.Pointers[target], ops
}

func (b *BTree[K, P, V]) putop(ops []Op[K, V]) error {
	log.Println("in putop")
	defer log.Println("done putop")

	ptr := b.Root

	var (
		path []P
		node *Node[K, P, V]
		err  error
	)

	for {
		if len(ops) > b.Order {
			panic("can't propagate too many ops")
		}

		path = append(path, ptr)
		node, err = b.cache.Get(ptr)
		if err != nil {
			return err
		}

		if node.isleaf() {
			for i := range ops {
				node.apply(&ops[i], b.compare)
			}
			node.Ops = nil

			if len(node.Values) > b.Order * 2 {
				log.Printf("added %d ops to a node with %d vals for a total of %d",
					len(ops), len(node.Values) - len(ops), len(node.Values))
				panic("invariant broken: more values than expected")
			}

			if len(node.Keys) < b.Order {
				return b.cache.Update(ptr, node)
			}

			new := node.split(b.compare)
			new.Next = node.Next
			newptr, err := b.cache.Put(new)
			if err != nil {
				return err
			}
			node.Next = &newptr
			if err := b.cache.Update(ptr, node); err != nil {
				return err
			}

			return b.insertUpwards(new.Keys[0], newptr, path[:len(path)-1])
		}

		log.Printf("about to append %d ops to a node with %d (order is %d)",
			len(ops), len(node.Ops), b.Order)
		node.Ops = append(node.Ops, ops...)
		if len(node.Ops) >= b.Order {
			// flush the ops down one level to the node
			// which should receive the most.
			var newptr P
			newptr, ops = b.pickforflush(node)

			if len(node.Ops) >= b.Order {
				panic("pickforflush wasn't enough")
			}

			if err := b.cache.Update(ptr, node); err != nil {
				return err
			}

			ptr = newptr
			continue
		}

		return b.cache.Update(ptr, node)
	}
}

func (b *BTree[K, P, V]) findleaf(key K) (node *Node[K, P, V], path []P, err error) {
	ptr := b.Root

	for {
		path = append(path, ptr)
		node, err = b.cache.Get(ptr)
		if err != nil {
			return
		}

		if node.isleaf() {
			return
		}

		idx, found := slices.BinarySearchFunc(node.Keys, key, b.compare)
		if found {
			idx++
		}
		if idx < len(node.Keys) {
			ptr = node.Pointers[idx]
		} else {
			ptr = node.Pointers[len(node.Keys)]
		}
	}
}

func (b *BTree[K, P, V]) Find(key K) (val V, found bool, err error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	leaf, _, err := b.findleaf(key)
	if err != nil {
		return
	}

	val, found = leaf.find(key, b.compare)
	return val, found, nil
}

func (n *Node[K, P, V]) find(key K, cmp func(K, K) int) (val V, found bool) {
	idx, found := slices.BinarySearchFunc(n.Keys, key, cmp)
	if found {
		return n.Values[idx], true
	}
	return val, false
}

func (n *Node[K, P, V]) insertAt(idx int, key K, val V) {
	n.Keys = slices.Insert(n.Keys, idx, key)
	n.Values = slices.Insert(n.Values, idx, val)
}

func (n *Node[K, P, V]) insertInternal(idx int, key K, ptr P) {
	// Pointers and Keys have different cardinalities, but to
	// decide whether to append or insert in Pointers we need
	// to consider the length of the keys.
	if idx >= len(n.Keys) {
		n.Keys = append(n.Keys, key)
		n.Pointers = append(n.Pointers, ptr)
		return
	}

	n.Keys = slices.Insert(n.Keys, idx, key)
	n.Pointers = slices.Insert(n.Pointers, idx+1, ptr)
}

func (b *BTree[K, P, V]) findsplit(key K, node *Node[K, P, V]) (int, bool) {
	return slices.BinarySearchFunc(node.Keys, key, b.compare)
}

func (node *Node[K, P, V]) apply(op *Op[K, V], cmp func(K, K) int) {
	// assume only OpAdd for now

	idx, has := slices.BinarySearchFunc(node.Keys, op.Key, cmp)
	if has {
		node.Values[idx] = op.Val
		return
	}
	node.insertAt(idx, op.Key, op.Val)
}

func (n *Node[K, P, V]) split(cmp func(K, K) int) (new *Node[K, P, V]) {
	cutoff := (len(n.Keys)+1)/2 - 1
	if cutoff == 0 {
		cutoff = 1
	}

	if n.isleaf() {
		new = newNodeFrom(n.Keys[cutoff:], []P{}, n.Values[cutoff:])
		n.Values = n.Values[:cutoff]
	} else {
		new = newNodeFrom(n.Keys[cutoff:], n.Pointers[cutoff+1:], []V{})
		n.Pointers = n.Pointers[:cutoff+1]
	}

	var nops []Op[K, V]
	for i := range n.Ops {
		if cmp(n.Ops[i].Key, new.Keys[0]) == -1 {
			nops = append(nops, n.Ops[i])
		} else {
			new.Ops = append(new.Ops, n.Ops[i])
		}
	}

	log.Println("new.keys[0] is", new.Keys[0], "and the split was")
	var a, b strings.Builder
	for i := range nops {
		fmt.Fprintf(&a, "%v ", nops[i].Key)
	}
	for i := range n.Ops {
		fmt.Fprintf(&a, "%v ", n.Ops[i].Key)
	}
	log.Println("node: ", a.String())
	log.Println("new:  ", b.String())

	n.Keys = n.Keys[:cutoff]
	return new
}

func (b *BTree[K, P, V]) Insert(key K, val V) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	op := Op[K, V]{
		Opcode: OpAdd,
		Key:    key,
		Val:    val,
	}
	return b.putop([]Op[K, V]{op})
}

func (b *BTree[K, P, V]) insertUpwards(key K, ptr P, path []P) error {
	log.Println("in insertUpwards")
	defer log.Println("done insertUpwards")

	for i := len(path) - 1; i >= 0; i-- {
		node, err := b.cache.Get(path[i])
		if err != nil {
			return err
		}

		idx, found := b.findsplit(key, node)
		if found {
			panic("broken invariant: duplicate key in intermediate node")
		}

		node.insertInternal(idx, key, ptr)
		if len(node.Keys) < b.Order {
			return b.cache.Update(path[i], node)
		}

		new := node.split(b.compare)
		key = new.Keys[0]
		new.Keys = new.Keys[1:]
		ptr, err = b.cache.Put(new)
		if err != nil {
			return err
		}
		if err := b.cache.Update(path[i], node); err != nil {
			return err
		}
	}

	log.Println("growing")

	// reached the root, growing the tree
	newroot := &Node[K, P, V]{
		Version:  versioning.FromString(NODE_VERSION),
		Keys:     []K{key},
		Pointers: []P{b.Root, ptr},
	}
	rootptr, err := b.cache.Put(newroot)
	if err != nil {
		return err
	}
	b.Root = rootptr
	return nil
}

func (b *BTree[K, P, V]) Stats() (hits, miss, size uint64) {
	return b.cache.lru.Stats()
}

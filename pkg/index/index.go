package index

import (
	"context"
	"sort"
	"sync"

	"github.com/google/btree"
	"github.com/regionless-storage-service/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Index interface {
	Get(ctx context.Context, key []byte, atRev int64) (rev, created Revision, ver int64, err error)
	Put(ctx context.Context, key []byte, rev Revision) error
	RangeSince(ctx context.Context, key, end []byte, rev int64) []Revision
	Tombstone(ctx context.Context, key []byte, rev Revision) error
	Equal(b Index) bool
}

type treeIndex struct {
	sync.RWMutex
	tree *btree.BTree
}

func NewTreeIndex() Index {
	return &treeIndex{
		tree: btree.New(32),
	}
}

// Put inserts a new index entry
// for now, assuming no error yield within this method
// todo: return applicable errors, if any
func (ti *treeIndex) Put(ctx context.Context, key []byte, rev Revision) error {
	// tracing indexing component - updating index
	_, span := otel.Tracer(config.TraceName).Start(ctx, "put index")
	defer span.End()

	keyi := &keyIndex{key: key}

	ti.Lock()
	defer ti.Unlock()
	item := ti.tree.Get(keyi)
	if item == nil {
		keyi.put(rev.main, rev.sub, rev.nodes)
		ti.tree.ReplaceOrInsert(keyi)
		return nil
	}
	okeyi := item.(*keyIndex)
	okeyi.put(rev.main, rev.sub, rev.nodes)
	return nil
}

func (ti *treeIndex) Restore(key []byte, created, modified Revision, ver int64) {
	keyi := &keyIndex{key: key}

	ti.Lock()
	defer ti.Unlock()
	item := ti.tree.Get(keyi)
	if item == nil {
		keyi.restore(created, modified, ver)
		ti.tree.ReplaceOrInsert(keyi)
		return
	}
	okeyi := item.(*keyIndex)
	okeyi.put(modified.main, modified.sub, modified.nodes)
}

func (ti *treeIndex) Get(ctx context.Context, key []byte, atRev int64) (modified, created Revision, ver int64, err error) {
	// tracing indexing component - lookup index
	_, span := otel.Tracer(config.TraceName).Start(ctx, "get index")
	defer span.End()

	keyi := &keyIndex{key: key}

	ti.RLock()
	defer ti.RUnlock()

	item := ti.tree.Get(keyi)
	if item == nil {
		span.RecordError(ErrRevisionNotFound)
		span.SetStatus(codes.Error, ErrRevisionNotFound.Error())
		return Revision{}, Revision{}, 0, ErrRevisionNotFound
	}

	keyi = item.(*keyIndex)
	if atRev == 0 {
		return keyi.get(int64(keyi.modified.main))
	}
	return keyi.get(atRev)
}

func (ti *treeIndex) Range(key, end []byte, atRev int64) (keys [][]byte, revs []Revision) {
	if end == nil {
		rev, _, _, err := ti.Get(context.TODO(), key, atRev)
		if err != nil {
			return nil, nil
		}
		return [][]byte{key}, []Revision{rev}
	}

	keyi := &keyIndex{key: key}
	endi := &keyIndex{key: end}

	ti.RLock()
	defer ti.RUnlock()

	ti.tree.AscendGreaterOrEqual(keyi, func(item btree.Item) bool {
		if len(endi.key) > 0 && !item.Less(endi) {
			return false
		}
		curKeyi := item.(*keyIndex)
		rev, _, _, err := curKeyi.get(atRev)
		if err != nil {
			return true
		}
		revs = append(revs, rev)
		keys = append(keys, curKeyi.key)
		return true
	})

	return keys, revs
}

func (ti *treeIndex) Tombstone(ctx context.Context, key []byte, rev Revision) error {
	// tracing indexing component - deleting index entry (as tombstone)
	_, span := otel.Tracer(config.TraceName).Start(ctx, "tombstone index")
	defer span.End()

	keyi := &keyIndex{key: key}

	ti.Lock()
	defer ti.Unlock()
	item := ti.tree.Get(keyi)
	if item == nil {
		return ErrRevisionNotFound
	}

	ki := item.(*keyIndex)
	return ki.tombstone(rev.main, rev.sub)
}

// RangeSince returns all Revisions from key(including) to end(excluding)
// at or after the given rev. The returned slice is sorted in the order
// of Revision.
func (ti *treeIndex) RangeSince(ctx context.Context, key, end []byte, rev int64) []Revision {
	// tracing indexing component - ranged query of  index
	_, span := otel.Tracer(config.TraceName).Start(ctx, "rangesince kv")
	defer span.End()

	ti.RLock()
	defer ti.RUnlock()

	keyi := &keyIndex{key: key}
	if end == nil {
		item := ti.tree.Get(keyi)
		if item == nil {
			return nil
		}
		keyi = item.(*keyIndex)
		return keyi.since(rev)
	}

	endi := &keyIndex{key: end}
	var revs []Revision
	ti.tree.AscendGreaterOrEqual(keyi, func(item btree.Item) bool {
		if len(endi.key) > 0 && !item.Less(endi) {
			return false
		}
		curKeyi := item.(*keyIndex)
		revs = append(revs, curKeyi.since(rev)...)
		return true
	})
	sort.Sort(Revisions(revs))

	return revs
}

func (a *treeIndex) Equal(bi Index) bool {
	b := bi.(*treeIndex)

	if a.tree.Len() != b.tree.Len() {
		return false
	}

	equal := true

	a.tree.Ascend(func(item btree.Item) bool {
		aki := item.(*keyIndex)
		bki := b.tree.Get(item).(*keyIndex)
		if !aki.equal(bki) {
			equal = false
			return false
		}
		return true
	})

	return equal
}

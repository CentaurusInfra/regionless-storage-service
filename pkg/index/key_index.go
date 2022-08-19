package index

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/google/btree"
	inc "github.com/regionless-storage-service/pkg/revision"
)

var (
	ErrRevisionNotFound = errors.New("mvcc: Revision not found")
)

// keyIndex stores the Revisions of a key in the backend.
// Each keyIndex has at least one key generation.
// Each generation might have several key versions.
// Tombstone on a key appends an tombstone version at the end
// of the current generation and creates a new empty generation.
// Each version of a key has an index pointing to the backend.
//
// For example: put(1.0);put(2.0);tombstone(3.0);put(4.0);tombstone(5.0) on key "foo"
// generate a keyIndex:
// key:     "foo"
// rev: 5
// generations:
//    {empty}
//    {4.0, 5.0(t)}
//    {1.0, 2.0, 3.0(t)}
//
// Compact a keyIndex removes the versions with smaller or equal to
// rev except the largest one. If the generation becomes empty
// during compaction, it will be removed. if all the generations get
// removed, the keyIndex should be removed.

// For example:
// compact(2) on the previous example
// generations:
//    {empty}
//    {4.0, 5.0(t)}
//    {2.0, 3.0(t)}
//
// compact(4)
// generations:
//    {empty}
//    {4.0, 5.0(t)}
//
// compact(5):
// generations:
//    {empty} -> key SHOULD be removed.
//
// compact(6):
// generations:
//    {empty} -> key SHOULD be removed.
type keyIndex struct {
	key         []byte
	modified    Revision // the main rev of the last modification
	generations []generation
}

// todo: return error instead to panic
// put puts a Revision to the keyIndex.
func (ki *keyIndex) put(main int64, sub int64, nodes []string) {
	rev := Revision{main: main, sub: sub, nodes: nodes}

	if !rev.GreaterThan(ki.modified) {
		panic(fmt.Errorf("store.keyindex: put with unexpected smaller Revision [%v / %v]", rev, ki.modified))
	}
	if len(ki.generations) == 0 {
		ki.generations = append(ki.generations, generation{})
	}
	g := &ki.generations[len(ki.generations)-1]
	if len(g.revs) == 0 { // create a new key
		g.created = Revision{main: int64(inc.GetGlobalIncreasingRevision())}
	}
	g.revs = append(g.revs, rev)
	g.ver++
	ki.modified = rev
}

func (ki *keyIndex) restore(created, modified Revision, ver int64) {
	if len(ki.generations) != 0 {
		panic("store.keyindex: cannot restore non-empty keyIndex")
	}

	ki.modified = modified
	g := generation{created: created, ver: ver, revs: []Revision{modified}}
	ki.generations = append(ki.generations, g)
}

// tombstone puts a Revision, pointing to a tombstone, to the keyIndex.
// It also creates a new empty generation in the keyIndex.
// It returns ErrRevisionNotFound when tombstone on an empty generation.
func (ki *keyIndex) tombstone(main int64, sub int64) error {
	if ki.isEmpty() {
		panic(fmt.Errorf("store.keyindex: unexpected tombstone on empty keyIndex %s", string(ki.key)))
	}
	if ki.generations[len(ki.generations)-1].isEmpty() {
		return ErrRevisionNotFound
	}
	ki.put(main, sub, nil)
	ki.generations = append(ki.generations, generation{})
	// keysGauge.Dec()
	return nil
}

// get gets the modified, created Revision and version of the key that satisfies the given atRev.
// Rev must be higher than or equal to the given atRev.
func (ki *keyIndex) get(atRev int64) (modified, created Revision, ver int64, err error) {
	if ki.isEmpty() {
		panic(fmt.Errorf("store.keyindex: unexpected get on empty keyIndex %s", string(ki.key)))
	}
	g := ki.findGeneration(atRev)
	if g.isEmpty() {
		return Revision{}, Revision{}, 0, ErrRevisionNotFound
	}

	n := g.walk(func(rev Revision) bool { return rev.main > atRev })
	if n != -1 {
		return g.revs[n], g.created, g.ver - int64(len(g.revs)-n-1), nil
	}

	return Revision{}, Revision{}, 0, ErrRevisionNotFound
}

// since returns Revisions since the given rev. Only the Revision with the
// largest sub Revision will be returned if multiple Revisions have the same
// main Revision.
func (ki *keyIndex) since(rev int64) []Revision {
	if ki.isEmpty() {
		panic(fmt.Errorf("store.keyindex: unexpected get on empty keyIndex %s", string(ki.key)))
	}
	since := Revision{rev, 0, nil}
	var gi int
	// find the generations to start checking
	for gi = len(ki.generations) - 1; gi > 0; gi-- {
		g := ki.generations[gi]
		if g.isEmpty() {
			continue
		}
		if since.GreaterThan(g.created) {
			break
		}
	}

	var revs []Revision
	var last int64
	for ; gi < len(ki.generations); gi++ {
		for _, r := range ki.generations[gi].revs {
			if since.GreaterThan(r) {
				continue
			}
			if r.main == last {
				// replace the Revision with a new one that has higher sub value,
				// because the original one should not be seen by external
				revs[len(revs)-1] = r
				continue
			}
			revs = append(revs, r)
			last = r.main
		}
	}
	return revs
}

func (ki *keyIndex) isEmpty() bool {
	return len(ki.generations) == 1 && ki.generations[0].isEmpty()
}

// findGeneration finds out the generation of the keyIndex that the
// given rev belongs to. If the given rev is at the gap of two generations,
// which means that the key does not exist at the given rev, it returns nil.
func (ki *keyIndex) findGeneration(rev int64) *generation {
	lastg := len(ki.generations) - 1
	cg := lastg

	for cg >= 0 {
		if len(ki.generations[cg].revs) == 0 {
			cg--
			continue
		}
		g := ki.generations[cg]
		if cg != lastg {
			if tomb := g.revs[len(g.revs)-1].main; tomb <= rev {
				return nil
			}
		}
		if g.revs[0].main <= rev {
			return &ki.generations[cg]
		}
		cg--
	}
	return nil
}

func (a *keyIndex) Less(b btree.Item) bool {
	return bytes.Compare(a.key, b.(*keyIndex).key) == -1
}

func (a *keyIndex) equal(b *keyIndex) bool {
	if !bytes.Equal(a.key, b.key) {
		return false
	}
	if a.modified.main != b.modified.main {
		return false
	}
	if a.modified.sub != b.modified.sub {
		return false
	}

	if len(a.generations) != len(b.generations) {
		return false
	}
	for i := range a.generations {
		ag, bg := a.generations[i], b.generations[i]
		if !ag.equal(bg) {
			return false
		}
	}
	return true
}

func (ki *keyIndex) String() string {
	var s string
	for _, g := range ki.generations {
		s += g.String()
	}
	return s
}

// generation contains multiple Revisions of a key.
type generation struct {
	ver     int64
	created Revision // when the generation is created (put in first Revision).
	revs    []Revision
}

func (g *generation) isEmpty() bool { return g == nil || len(g.revs) == 0 }

// walk walks through the Revisions in the generation in descending order.
// It passes the Revision to the given function.
// walk returns until: 1. it finishes walking all pairs 2. the function returns false.
// walk returns the position at where it stopped. If it stopped after
// finishing walking, -1 will be returned.
func (g *generation) walk(f func(rev Revision) bool) int {
	l := len(g.revs)
	for i := range g.revs {
		ok := f(g.revs[l-i-1])
		if !ok {
			return l - i - 1
		}
	}
	return -1
}

func (g *generation) String() string {
	return fmt.Sprintf("g: created[%d] ver[%d], revs %#v\n", g.created, g.ver, g.revs)
}

func (a generation) equal(b generation) bool {
	if a.ver != b.ver {
		return false
	}
	if len(a.revs) != len(b.revs) {
		return false
	}

	for i := range a.revs {
		ar, br := a.revs[i], b.revs[i]
		if ar.main != br.main && ar.sub != br.sub {
			return false
		}
	}
	return true
}

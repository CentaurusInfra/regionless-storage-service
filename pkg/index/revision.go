// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package index

import (
	"encoding/binary"
	"fmt"
)

// revBytesLen is the byte length of a normal Revision.
// First 8 bytes is the Revision.main in big-endian format. The 9th byte
// is a '_'. The last 8 bytes is the Revision.sub in big-endian format.
const revBytesLen = 8 + 1 + 8

// A Revision indicates modification of the key-value space.
// The set of changes that share same main Revision changes the key-value space atomically.
type Revision struct {
	// main is the main Revision of a set of changes that happen atomically.
	main int64

	// sub is the the sub Revision of a change in a set of changes that happen
	// atomically. Each change has different increasing sub Revision in that
	// set.
	sub int64

	nodes []string
}

func NewRevision(main, sub int64, nodes []string) Revision {
	return Revision{main: main, sub: sub, nodes: nodes}
}
func (a Revision) String() string {
	return fmt.Sprintf("%d", a.main)
}
func (a Revision) GetMain() int64 {
	return a.main
}
func (a Revision) GetSub() int64 {
	return a.sub
}
func (a Revision) GetNodes() []string {
	return a.nodes
}
func (a *Revision) SetNodes(ns []string) {
	a.nodes = ns
}
func (a Revision) GreaterThan(b Revision) bool {
	if a.main > b.main {
		return true
	}
	if a.main < b.main {
		return false
	}
	return a.sub > b.sub
}

func revToBytes(rev Revision, bytes []byte) {
	binary.BigEndian.PutUint64(bytes, uint64(rev.main))
	bytes[8] = '_'
	binary.BigEndian.PutUint64(bytes[9:], uint64(rev.sub))
}

func bytesToRev(bytes []byte) Revision {
	return Revision{
		main: int64(binary.BigEndian.Uint64(bytes[0:8])),
		sub:  int64(binary.BigEndian.Uint64(bytes[9:])),
	}
}

type Revisions []Revision

func (a Revisions) Len() int           { return len(a) }
func (a Revisions) Less(i, j int) bool { return a[j].GreaterThan(a[i]) }
func (a Revisions) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

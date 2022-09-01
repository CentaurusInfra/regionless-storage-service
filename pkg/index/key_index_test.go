package index

import (
	"reflect"
	"testing"

	"github.com/regionless-storage-service/pkg/partition/consistent"
)

func TestPut(t *testing.T) {
	tcs := []struct {
		name                string
		index               keyIndex
		revToPut            Revision
		expectedModified    Revision
		expectedGenerations []generation
	}{
		{
			name: "put newer rev",
			index: keyIndex{
				key:      []byte("testkey"),
				modified: Revision{main: 99, sub: 0, nodes: []consistent.RkvNode{{Name: "node2"}}},
				generations: []generation{{
					ver:     2,
					created: Revision{main: 98, sub: 0, nodes: []consistent.RkvNode{{Name: "node1"}}},
					revs: []Revision{
						{main: 98, sub: 0, nodes: []consistent.RkvNode{{Name: "node1"}}},
						{main: 99, sub: 0, nodes: []consistent.RkvNode{{Name: "node2"}}},
					},
				}}},
			revToPut:         Revision{main: 100, sub: 0, nodes: []consistent.RkvNode{{Name: "node3"}}},
			expectedModified: Revision{main: 100, sub: 0, nodes: []consistent.RkvNode{{Name: "node3"}}},
			expectedGenerations: []generation{{
				ver:     3,
				created: Revision{main: 98, sub: 0, nodes: []consistent.RkvNode{{Name: "node1"}}},
				revs: []Revision{
					{main: 98, sub: 0, nodes: []consistent.RkvNode{{Name: "node1"}}},
					{main: 99, sub: 0, nodes: []consistent.RkvNode{{Name: "node2"}}},
					{main: 100, sub: 0, nodes: []consistent.RkvNode{{Name: "node3"}}},
				},
			},
			},
		},
		{
			name: "put stale rev",
			index: keyIndex{
				key:      []byte("testkey"),
				modified: Revision{main: 100, sub: 0, nodes: []consistent.RkvNode{{Name: "node3"}}},
				generations: []generation{{
					ver:     2,
					created: Revision{main: 98, sub: 0, nodes: []consistent.RkvNode{{Name: "node1"}}},
					revs: []Revision{
						{main: 98, sub: 0, nodes: []consistent.RkvNode{{Name: "node1"}}},
						{main: 100, sub: 0, nodes: []consistent.RkvNode{{Name: "node3"}}},
					},
				}}},
			revToPut:         Revision{main: 99, sub: 0, nodes: []consistent.RkvNode{{Name: "node2"}}},
			expectedModified: Revision{main: 100, sub: 0, nodes: []consistent.RkvNode{{Name: "node3"}}},
			expectedGenerations: []generation{{
				ver:     2,
				created: Revision{main: 98, sub: 0, nodes: []consistent.RkvNode{{Name: "node1"}}},
				revs: []Revision{
					{main: 98, sub: 0, nodes: []consistent.RkvNode{{Name: "node1"}}},
					{main: 99, sub: 0, nodes: []consistent.RkvNode{{Name: "node2"}}},
					{main: 100, sub: 0, nodes: []consistent.RkvNode{{Name: "node3"}}},
				},
			},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.index.put(tc.revToPut.main, tc.revToPut.sub, tc.revToPut.GetNodes())

			if !reflect.DeepEqual(tc.index.modified, tc.expectedModified) {
				t.Errorf("extecped modified rev %v, got %v", tc.expectedModified, tc.index.modified)
			}

			if !reflect.DeepEqual(tc.expectedGenerations, tc.index.generations) {
				t.Errorf("extecped resultant generations %v, got %v", tc.expectedGenerations, tc.index.generations)
			}
		})
	}
}

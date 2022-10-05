package index

import (
	"reflect"
	"testing"
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
				modified: Revision{main: 99, sub: 0, nodes: []string{"node2"}},
				generations: []generation{{
					ver:     2,
					created: Revision{main: 98, sub: 0, nodes: []string{"node1"}},
					revs: []Revision{
						{main: 98, sub: 0, nodes: []string{"node1"}},
						{main: 99, sub: 0, nodes: []string{"node2"}},
					},
				}}},
			revToPut:         Revision{main: 100, sub: 0, nodes: []string{"node3"}},
			expectedModified: Revision{main: 100, sub: 0, nodes: []string{"node3"}},
			expectedGenerations: []generation{{
				ver:     3,
				created: Revision{main: 98, sub: 0, nodes: []string{"node1"}},
				revs: []Revision{
					{main: 98, sub: 0, nodes: []string{"node1"}},
					{main: 99, sub: 0, nodes: []string{"node2"}},
					{main: 100, sub: 0, nodes: []string{"node3"}},
				},
			},
			},
		},
		{
			name: "put stale rev",
			index: keyIndex{
				key:      []byte("testkey"),
				modified: Revision{main: 100, sub: 0, nodes: []string{"node3"}},
				generations: []generation{{
					ver:     2,
					created: Revision{main: 98, sub: 0, nodes: []string{"node1"}},
					revs: []Revision{
						{main: 98, sub: 0, nodes: []string{"node1"}},
						{main: 100, sub: 0, nodes: []string{"node3"}},
					},
				}}},
			revToPut:         Revision{main: 99, sub: 0, nodes: []string{"node2"}},
			expectedModified: Revision{main: 100, sub: 0, nodes: []string{"node3"}},
			expectedGenerations: []generation{{
				ver:     2,
				created: Revision{main: 98, sub: 0, nodes: []string{"node1"}},
				revs: []Revision{
					{main: 98, sub: 0, nodes: []string{"node1"}},
					{main: 99, sub: 0, nodes: []string{"node2"}},
					{main: 100, sub: 0, nodes: []string{"node3"}},
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

func TestUpdate(t *testing.T) {
	tcs := []struct {
		name                string
		index               keyIndex
		revToPut            Revision
		revToAssume         int64
		expectedModified    Revision
		expectedGenerations []generation
		expectedError       string
	}{
		{
			name: "update rev on top of assumed one and succeed",
			index: keyIndex{
				key:      []byte("testkey"),
				modified: Revision{main: 99, sub: 0, nodes: []string{"node2"}},
				generations: []generation{{
					ver:     2,
					created: Revision{main: 98, sub: 0, nodes: []string{"node1"}},
					revs: []Revision{
						{main: 98, sub: 0, nodes: []string{"node1"}},
						{main: 99, sub: 0, nodes: []string{"node2"}},
					},
				}}},
			revToPut:         Revision{main: 100, sub: 0, nodes: []string{"node3"}},
			revToAssume:      99,
			expectedModified: Revision{main: 100, sub: 0, nodes: []string{"node3"}},
			expectedGenerations: []generation{{
				ver:     3,
				created: Revision{main: 98, sub: 0, nodes: []string{"node1"}},
				revs: []Revision{
					{main: 98, sub: 0, nodes: []string{"node1"}},
					{main: 99, sub: 0, nodes: []string{"node2"}},
					{main: 100, sub: 0, nodes: []string{"node3"}},
				},
			},
			},
		},
		{
			name: "update stale rev and fail",
			index: keyIndex{
				key:      []byte("testkey"),
				modified: Revision{main: 100, sub: 0, nodes: []string{"node3"}},
				generations: []generation{{
					ver:     2,
					created: Revision{main: 98, sub: 0, nodes: []string{"node1"}},
					revs: []Revision{
						{main: 98, sub: 0, nodes: []string{"node1"}},
						{main: 100, sub: 0, nodes: []string{"node3"}},
					},
				}}},
			revToPut:         Revision{main: 101, sub: 0, nodes: []string{"node2"}},
			revToAssume:      98,
			expectedModified: Revision{main: 100, sub: 0, nodes: []string{"node3"}},
			expectedGenerations: []generation{
				{
					ver:     2,
					created: Revision{main: 98, sub: 0, nodes: []string{"node1"}},
					revs: []Revision{
						{main: 98, sub: 0, nodes: []string{"node1"}},
						{main: 100, sub: 0, nodes: []string{"node3"}},
					},
				},
			},
			expectedError: "the rev to assume is not the latest one",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.index.update(tc.revToPut.main, tc.revToPut.sub, tc.revToPut.GetNodes(), tc.revToAssume)

			if len(tc.expectedError) == 0 && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(tc.expectedError) != 0 {
				if err == nil {
					t.Errorf("test case should have failed but not")
				}

				if tc.expectedError != err.Error() {
					t.Errorf("unexpected error message: %s", err)
				}
			}

			if !reflect.DeepEqual(tc.index.modified, tc.expectedModified) {
				t.Errorf("extecped modified rev %v, got %v", tc.expectedModified, tc.index.modified)
			}

			if !reflect.DeepEqual(tc.expectedGenerations, tc.index.generations) {
				t.Errorf("extecped resultant generations %v, got %v", tc.expectedGenerations, tc.index.generations)
			}
		})
	}
}

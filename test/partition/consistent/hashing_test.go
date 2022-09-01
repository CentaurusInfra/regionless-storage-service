package consistent

import (
	"strings"
	"testing"

	"github.com/regionless-storage-service/pkg/constants"
	"github.com/regionless-storage-service/pkg/partition/consistent"
)

func TestNewSyncAsyncHashingManager(t *testing.T) {
	localStores := map[constants.AvailabilityZone][]consistent.RkvNode{
		constants.US_EAST_1A: {{Name: "1.1.3.4:1000"}, {Name: "1.2.3.4:1000"}},
		constants.US_EAST_2A: {{Name: "3.2.3.4:2000"}, {Name: "3.1.3.4:2000"}, {Name: "3.9.3.4:2000"}},
		constants.US_WEST_1A: {{Name: "6.2.3.4:3000"}, {Name: "6.2.3.4:3000"}},
	}

	RemoteStores := []consistent.RkvNode{{Name: "9.2.3.4:1000"}, {Name: "9.2.3.4:2000"}}

	h := consistent.NewSyncAsyncHashingManager(constants.Rendezvous, localStores, 2, RemoteStores, 1)
	l, err := h.GetSyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if len(l) != 2 {
		t.Fatalf("Local nodes shouldn't be %d", len(l))
	}
	if l == nil {
		t.Fatalf("Local nodes shouldn't be nil: %v", l)
	}
	r, err := h.GetAsyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if r == nil {
		t.Fatalf("Remote nodes shouldn't be nil: %v", r)
	}
	if len(r) != 1 {
		t.Fatalf("Remote nodes shouldn't be %d", len(r))
	}
	lnMap := make(map[string]bool)
	for _, ln := range l {
		lk := ln.String()[0:1]
		if _, found := lnMap[lk]; found {
			t.Fatalf("Same az selected %v", ln)
		}
		lnMap[lk] = true
	}
	for _, rn := range r {
		rk := rn.String()[0:1]
		if rk != "9" {
			t.Fatalf("Unexpected az selected %v", rn)
		}
	}
}

func TestOneLocalStore(t *testing.T) {
	localStores := map[constants.AvailabilityZone][]consistent.RkvNode{
		constants.US_EAST_1A: {{Name: "1.1.3.4:1000"}, {Name: "1.2.3.4:1000"}},
		constants.US_EAST_2A: {{Name: "3.2.3.4:2000"}, {Name: "3.1.3.4:2000"}, {Name: "3.9.3.4:2000"}},
		constants.US_WEST_1A: {{Name: "6.2.3.4:3000"}, {Name: "6.2.3.4:3000"}},
	}

	RemoteStores := []consistent.RkvNode{{Name: "9.2.3.4:1000"}, {Name: "9.2.3.4:2000"}}

	h := consistent.NewSyncAsyncHashingManager(constants.Rendezvous, localStores, 1, RemoteStores, 0)
	l, err := h.GetSyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if len(l) != 1 {
		t.Fatalf("Local nodes shouldn't be %d", len(l))
	}
	if l == nil {
		t.Fatalf("Local nodes shouldn't be nil: %v", l)
	}
	r, err := h.GetAsyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if r == nil {
		t.Fatalf("Remote nodes shouldn't be nil: %v", r)
	}
	if len(r) != 0 {
		t.Fatalf("Remote nodes shouldn't be %d", len(r))
	}
}

func TestZeroRemoteStore(t *testing.T) {
	localStores := map[constants.AvailabilityZone][]consistent.RkvNode{
		constants.US_EAST_1A: {{Name: "1.1.3.4:1000"}, {Name: "1.2.3.4:1000"}},
		constants.US_EAST_2A: {{Name: "3.2.3.4:2000"}, {Name: "3.1.3.4:2000"}, {Name: "3.9.3.4:2000"}},
		constants.US_WEST_1A: {{Name: "6.2.3.4:3000"}, {Name: "6.2.3.4:3000"}},
	}

	RemoteStores := []consistent.RkvNode{}

	h := consistent.NewSyncAsyncHashingManager(constants.Rendezvous, localStores, 1, RemoteStores, 1)
	l, err := h.GetSyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if len(l) != 1 {
		t.Fatalf("Local nodes shouldn't be %d", len(l))
	}
	if l == nil {
		t.Fatalf("Local nodes shouldn't be nil: %v", l)
	}
	_, err = h.GetAsyncNodes([]byte("1"))
	if err == nil {
		t.Fatal("Error should be expected")
	}
}

func TestGetSortedSyncStores(t *testing.T) {
	localStores := map[constants.AvailabilityZone][]consistent.RkvNode{
		constants.US_EAST_1A: {{Name: "1.1.3.4:1000", Latency: 100}, {Name: "1.2.3.4:1000", Latency: 101}},
		constants.US_EAST_2A: {{Name: "3.2.3.4:2000", Latency: 201}, {Name: "3.1.3.4:2000", Latency: 203}, {Name: "3.9.3.4:2000", Latency: 202}},
		constants.US_WEST_1A: {{Name: "6.2.3.4:3000", Latency: 301}, {Name: "6.2.3.4:3000", Latency: 303}},
	}

	RemoteStores := []consistent.RkvNode{{Name: "9.2.3.4:1000"}, {Name: "9.2.3.4:2000"}}

	h := consistent.NewSyncAsyncHashingManager(constants.Rendezvous, localStores, 3, RemoteStores, 1)
	l, err := h.GetSyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if len(l) != 3 {
		t.Fatalf("Local nodes shouldn't be %d", len(l))
	}
	for i := 0; i < len(l)-1; i++ {
		if h.LatencyMap[l[i].String()] > h.LatencyMap[l[i+1].String()] {
			t.Fatalf("The nodes is not sorted by latency")
		}
	}
	r, err := h.GetAsyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if r == nil {
		t.Fatalf("Remote nodes shouldn't be nil: %v", r)
	}
	if len(r) != 1 {
		t.Fatalf("Remote node size shouldn't be %d", len(r))
	}
	nodes, err := h.GetNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("Node size shouldn't be %d", len(nodes))
	}
	if nodes[1] != r[0].String() {
		t.Fatalf("Remote node shouldn't be %s", nodes[1])
	}
	sortedSyncNodes := strings.Split(nodes[0], ",")
	for i := 0; i < len(l); i++ {
		if sortedSyncNodes[i] != l[i].String() {
			t.Fatalf("stored node shouldn't be %s", sortedSyncNodes[i])
		}
	}
}

func TestNewSyncHashingManager(t *testing.T) {
	localStores := []consistent.RkvNode{
		{Name: "1.1.3.4:1000"}, {Name: "1.2.3.4:1000"},
		{Name: "3.2.3.4:2000"}, {Name: "3.1.3.4:2000"}, {Name: "3.9.3.4:2000"},
		{Name: "6.2.3.4:3000"}, {Name: "6.2.3.4:3000"},
	}

	h := consistent.NewSyncHashingManager(constants.Rendezvous, localStores, 1)
	l, err := h.GetSyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if l == nil {
		t.Fatalf("Local nodes shouldn't be nil: %v", l)
	}
	if len(l) != 1 {
		t.Fatalf("Local nodes shouldn't be %d", len(l))
	}
	r, err := h.GetAsyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if r != nil {
		t.Fatalf("Remote nodes should be nil: %v", r)
	}
	n, err := h.GetNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if len(n) != 1 {
		t.Fatalf("Local nodes shouldn't be %d", len(n))
	}
	if n[0] != l[0].String() {
		t.Fatalf("Local nodes shouldn't be %d", len(n))
	}
}

func TestNewSyncHashingManagerGetMutipleNodes(t *testing.T) {
	localStores := []consistent.RkvNode{
		{Name: "1.1.3.4:1000"}, {Name: "1.2.3.4:1000"},
		{Name: "3.2.3.4:2000"}, {Name: "3.1.3.4:2000"}, {Name: "3.9.3.4:2000"},
		{Name: "6.2.3.4:3000"}, {Name: "6.2.3.4:3000"},
	}

	h := consistent.NewSyncHashingManager(constants.Rendezvous, localStores, 2)
	l, err := h.GetSyncNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if l == nil {
		t.Fatalf("Local nodes shouldn't be nil: %v", l)
	}
	if len(l) != 2 {
		t.Fatalf("Local nodes shouldn't be %d", len(l))
	}

	n, err := h.GetNodes([]byte("1"))
	if err != nil {
		t.Fatalf("Get unexpected error %v", err)
	}
	if len(n) != 1 {
		t.Fatalf("Local nodes shouldn't be %d", len(n))
	}
	if n[0] != l[0].String()+","+l[1].String() {
		t.Fatalf("Local nodes shouldn't be %d", len(n))
	}
}

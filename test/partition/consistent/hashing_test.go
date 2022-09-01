package consistent

import (
	"testing"

	"github.com/regionless-storage-service/pkg/constants"
	"github.com/regionless-storage-service/pkg/partition/consistent"
)

func TestNewHashingWithLocalAndRemote(t *testing.T) {
	localStores := map[constants.AvailabilityZone][]consistent.RkvNode{
		constants.US_EAST_1A: {{Name: "1.1.3.4:1000"}, {Name: "1.2.3.4:1000"}},
		constants.US_EAST_2A: {{Name: "3.2.3.4:2000"}, {Name: "3.1.3.4:2000"}, {Name: "3.9.3.4:2000"}},
		constants.US_WEST_1A: {{Name: "6.2.3.4:3000"}, {Name: "6.2.3.4:3000"}},
	}

	RemoteStores := []consistent.RkvNode{{Name: "9.2.3.4:1000"}, {Name: "9.2.3.4:2000"}}

	h := consistent.NewHashingWithLocalAndRemote(localStores, 2, RemoteStores, 1)
	l, r, _ := h.GetLocalAndRemoteNodes([]byte("1"))
	if l == nil {
		t.Fatalf("Local nodes shouldn't be nil: %v", l)
	}
	if r == nil {
		t.Fatalf("Remote nodes shouldn't be nil: %v", r)
	}
	if len(l) != 2 {
		t.Fatalf("Local nodes shouldn't be %d", len(l))
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

	h := consistent.NewHashingWithLocalAndRemote(localStores, 1, RemoteStores, 0)
	l, r, _ := h.GetLocalAndRemoteNodes([]byte("1"))
	if l == nil {
		t.Fatalf("Local nodes shouldn't be nil: %v", l)
	}
	if r == nil {
		t.Fatalf("Remote nodes shouldn't be nil: %v", r)
	}
	if len(l) != 1 {
		t.Fatalf("Local nodes shouldn't be %d", len(l))
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

	h := consistent.NewHashingWithLocalAndRemote(localStores, 1, RemoteStores, 1)
	_, _, err := h.GetLocalAndRemoteNodes([]byte("1"))
	if err == nil {
		t.Fatalf("err is expected")
	}
}

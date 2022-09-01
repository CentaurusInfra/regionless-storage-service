package consistent

import (
	"fmt"
	"time"

	"github.com/cespare/xxhash"
	"github.com/regionless-storage-service/pkg/constants"
)

type Hasher interface {
	Hash([]byte) uint64
}

type Node interface {
	String() string
}

type ConsistentHashing interface {
	AddNode(node Node)
	LocateKey(key []byte) Node
	LocateNodes(key []byte, count int) []Node
}

type RkvNode struct {
	Name     string
	Latency  time.Duration
	IsRemote bool
}

func (tn RkvNode) String() string {
	return tn.Name
}

type rkvHash struct{}

func (th rkvHash) Hash(key []byte) uint64 {
	return xxhash.Sum64(key)
}

type KV struct {
	key, value string
}

type HashingWithAzandRemote struct {
	AzHashing    ConsistentHashing
	LocalHashing map[constants.AvailabilityZone]ConsistentHashing
	RemoteHasing ConsistentHashing
	LocalCount   int
	RemoteCount  int
}

func NewHashingWithLocalAndRemote(localStores map[constants.AvailabilityZone][]RkvNode, localCount int, remoteStores []RkvNode, remoteCount int) HashingWithAzandRemote {
	hasher := rkvHash{}
	azRing := NewRendezvous(nil, hasher)
	localRing := make(map[constants.AvailabilityZone]ConsistentHashing)
	for az, stores := range localStores {
		azRing.AddNode(RkvNode{Name: az.Name()})
		if _, found := localRing[az]; !found {
			localRing[az] = NewRendezvous(nil, hasher)
		}
		for _, store := range stores {
			localRing[az].AddNode(store)
		}
	}
	remoteRing := NewRendezvous(nil, hasher)
	for _, store := range remoteStores {
		remoteRing.AddNode(store)
	}
	return HashingWithAzandRemote{AzHashing: azRing, LocalHashing: localRing, RemoteHasing: remoteRing, LocalCount: localCount, RemoteCount: remoteCount}
}

func (h HashingWithAzandRemote) GetLocalAndRemoteNodes(key []byte) ([]RkvNode, []RkvNode, error) {
	localNodes := []RkvNode{}
	azs := h.AzHashing.LocateNodes(key, h.LocalCount)
	if len(azs) != h.LocalCount {
		return nil, nil, fmt.Errorf("failed to get %d zones. The return number is %d", h.LocalCount, len(azs))
	}
	for _, az := range azs {
		lnodes := h.LocalHashing[constants.AvailabilityZone(az.String())].LocateNodes(key, 1)
		if len(lnodes) != 1 {
			return nil, nil, fmt.Errorf("failed to get 1 local node. The return number is %d", len(lnodes))
		}
		localNodes = append(localNodes, RkvNode{Name: lnodes[0].String()})
	}
	remoteNodes := []RkvNode{}
	rnodes := h.RemoteHasing.LocateNodes(key, h.RemoteCount)
	if len(rnodes) != h.RemoteCount {
		return nil, nil, fmt.Errorf("failed to get %d remote nodes. The return number is %d", h.RemoteCount, len(rnodes))
	}
	for _, rn := range rnodes {
		remoteNodes = append(remoteNodes, RkvNode{Name: rn.String()})
	}
	return localNodes, remoteNodes, nil
}

package consistent

import (
	"fmt"
	"sort"
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

func (rn RkvNode) String() string {
	return rn.Name
}

type rkvHash struct{}

func (th rkvHash) Hash(key []byte) uint64 {
	return xxhash.Sum64(key)
}

type KV struct {
	key, value string
}

type HashingManager interface {
	GetSyncNodes(key []byte) ([]Node, error)
	GetAsyncNodes(key []byte) ([]Node, error)
}

type SyncHashingManager struct {
	hasing ConsistentHashing
	count  int
}

func NewSyncHashingManager(hashingType constants.ConsistentHashingType, nodes []RkvNode, count int) SyncHashingManager {
	h := Factory(hashingType)
	for _, node := range nodes {
		h.AddNode(node)
	}
	return SyncHashingManager{hasing: h, count: count}
}

func (fhm SyncHashingManager) GetSyncNodes(key []byte) ([]Node, error) {
	return fhm.hasing.LocateNodes(key, fhm.count), nil
}

func (fhm SyncHashingManager) GetAsyncNodes(key []byte) ([]Node, error) {
	return nil, nil
}

type SyncByZoneAsyncHashingManager struct {
	AzHashing    ConsistentHashing
	LocalHashing map[constants.AvailabilityZone]ConsistentHashing
	RemoteHasing ConsistentHashing
	LatencyMap   map[string]time.Duration
	LocalCount   int
	RemoteCount  int
}

func NewSyncAsyncHashingManager(hashingType constants.ConsistentHashingType, localStores map[constants.AvailabilityZone][]RkvNode, localCount int, remoteStores []RkvNode, remoteCount int) SyncByZoneAsyncHashingManager {
	azRing := Factory(hashingType)
	localRing := make(map[constants.AvailabilityZone]ConsistentHashing)
	latencyMap := make(map[string]time.Duration)
	for az, stores := range localStores {
		azRing.AddNode(RkvNode{Name: az.Name()})
		if _, found := localRing[az]; !found {
			localRing[az] = Factory(hashingType)
		}
		for _, store := range stores {
			localRing[az].AddNode(store)
			latencyMap[store.Name] = store.Latency
		}
	}
	remoteRing := Factory(hashingType)
	for _, store := range remoteStores {
		remoteRing.AddNode(store)
		latencyMap[store.Name] = store.Latency
	}
	return SyncByZoneAsyncHashingManager{AzHashing: azRing, LocalHashing: localRing, RemoteHasing: remoteRing, LocalCount: localCount, RemoteCount: remoteCount, LatencyMap: latencyMap}
}

func (sahm SyncByZoneAsyncHashingManager) GetSyncNodes(key []byte) ([]Node, error) {
	nodesWithLatency := make([]RkvNode, 0)
	azs := sahm.AzHashing.LocateNodes(key, sahm.LocalCount)
	if len(azs) != sahm.LocalCount {
		return nil, fmt.Errorf("failed to get %d zones. The return number is %d", sahm.LocalCount, len(azs))
	}
	for _, az := range azs {
		lnodes := sahm.LocalHashing[constants.AvailabilityZone(az.String())].LocateNodes(key, 1)
		if len(lnodes) != 1 {
			return nil, fmt.Errorf("failed to get 1 local node. The return number is %d", len(lnodes))
		}
		nodesWithLatency = append(nodesWithLatency, RkvNode{Name: lnodes[0].String(), Latency: sahm.LatencyMap[lnodes[0].String()]})

	}
	sort.Slice(nodesWithLatency, func(i, j int) bool {
		return nodesWithLatency[i].Latency < nodesWithLatency[j].Latency
	})

	localNodes := make([]Node, 0)
	for _, node := range nodesWithLatency {
		localNodes = append(localNodes, node)
	}
	return localNodes, nil
}

func (sahm SyncByZoneAsyncHashingManager) GetAsyncNodes(key []byte) ([]Node, error) {
	rnodes := sahm.RemoteHasing.LocateNodes(key, sahm.RemoteCount)
	if len(rnodes) != sahm.RemoteCount {
		return nil, fmt.Errorf("failed to get %d remote nodes. The return number is %d", sahm.RemoteCount, len(rnodes))
	}
	return rnodes, nil
}

func Factory(hashingType constants.ConsistentHashingType) ConsistentHashing {
	switch hashingType {
	case constants.Rendezvous:
		return NewRendezvous(nil, rkvHash{})
	case constants.Ring:
		return NewRingHashing(rkvHash{})
	default:
		return NewRendezvous(nil, rkvHash{})
	}
}

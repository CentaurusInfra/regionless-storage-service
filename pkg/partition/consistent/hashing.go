package consistent

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cespare/xxhash"
	"github.com/regionless-storage-service/pkg/constants"
	"k8s.io/klog"
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

type HashingManager interface {
	GetSyncNodes(key []byte) ([]Node, error)
	GetAsyncNodes(key []byte) ([]Node, error)
	GetNodes(key []byte) ([]string, error)
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

func (shm SyncHashingManager) GetSyncNodes(key []byte) ([]Node, error) {
	return shm.hasing.LocateNodes(key, shm.count), nil
}

func (shm SyncHashingManager) GetAsyncNodes(key []byte) ([]Node, error) {
	return nil, nil
}

func (shm SyncHashingManager) GetNodes(key []byte) ([]string, error) {
	syncNodes, err := shm.GetSyncNodes(key)
	res := make([]string, 0)
	if err != nil {
		klog.Errorf("failed to get all the sync nodes: %v", err)
		return res, err
	}
	return []string{strings.Join(convertNodeArrToStringArr(syncNodes), ",")}, nil
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
	localNodes := make([]Node, 0)
	if sahm.LocalCount < 1 {
		return localNodes, nil
	}
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
	for _, node := range nodesWithLatency {
		localNodes = append(localNodes, node)
	}
	return localNodes, nil
}

func (sahm SyncByZoneAsyncHashingManager) GetAsyncNodes(key []byte) ([]Node, error) {
	if sahm.RemoteCount < 1 {
		return make([]Node, 0), nil
	}
	rnodes := sahm.RemoteHasing.LocateNodes(key, sahm.RemoteCount)
	if len(rnodes) != sahm.RemoteCount {
		return nil, fmt.Errorf("failed to get %d remote nodes. The return number is %d", sahm.RemoteCount, len(rnodes))
	}
	return rnodes, nil
}

func (sahm SyncByZoneAsyncHashingManager) GetNodes(key []byte) ([]string, error) {
	syncNodes, err := sahm.GetSyncNodes(key)
	res := make([]string, 0)
	if err != nil {
		klog.Errorf("failed to get all the sync nodes: %v", err)
		return res, err
	}
	syncNodesString := strings.Join(convertNodeArrToStringArr(syncNodes), ",")
	asyncNodes, err := sahm.GetAsyncNodes(key)
	if err != nil {
		klog.Errorf("failed to get all the async nodes: %v", err)
		return res, err
	}
	asyncNodesString := strings.Join(convertNodeArrToStringArr(asyncNodes), ",")
	return []string{syncNodesString, asyncNodesString}, nil
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

func convertNodeArrToStringArr(nodes []Node) []string {
	res := make([]string, 0)
	for _, node := range nodes {
		res = append(res, node.String())
	}
	return res
}

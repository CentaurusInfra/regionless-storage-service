package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/regionless-storage-service/pkg/constants"
	"github.com/regionless-storage-service/pkg/network/latency"
	"github.com/regionless-storage-service/pkg/partition/consistent"
)

const (
	TraceName       = "regionless-kv-store"
	DefaultTraceEnv = "rkv-test"
)

var (
	TraceEnv          string
	TraceSamplingRate float64

	// RKVConfig keeps rkv configuration parsed from config.json
	RKVConfig *KVConfiguration
)

type KVConfiguration struct {
	ConsistentHash                        constants.ConsistentHashingType
	StoreType                             constants.StoreType
	HashingManagerType                    constants.HashingManagerType
	PipeType                              constants.PipingType
	Stores                                []KVStore
	BucketSize                            int64
	ReplicaNum                            ReplicaNum
	Concurrent                            bool
	RemoteStoreLatencyThresholdInMilliSec int64
}

type ReplicaNum struct {
	Local  int
	Remote int
}

type KVStore struct {
	Region                constants.Region
	AvailabilityZone      constants.AvailabilityZone
	Name                  string
	Host                  string
	Port                  int
	ArtificialLatencyInMs int
}

func NewKVConfiguration(fileName string) (*KVConfiguration, error) {
	_, runningfile, _, ok := runtime.Caller(1)
	configuration := &KVConfiguration{}
	if !ok {
		return configuration, fmt.Errorf("failed to open the given config file %s", fileName)
	}
	filepath := path.Join(path.Dir(runningfile), fileName)
	file, err := os.Open(filepath)
	if err != nil {
		return configuration, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)

	err = decoder.Decode(&configuration)
	return configuration, err
}

// Returns local stores grouping by AvailabilityZone and remote stores in array
func (c *KVConfiguration) GetReplications() (map[constants.AvailabilityZone][]consistent.RkvNode, []consistent.RkvNode, error) {
	localStores := make(map[constants.AvailabilityZone][]consistent.RkvNode)
	remoteStores := make([]consistent.RkvNode, 0)
	for _, store := range c.Stores {
		target := fmt.Sprintf("%s:%d", store.Host, store.Port)
		storeLatency := int64(0)
		if c.StoreType == constants.DummyLatency {
			storeLatency = int64(store.ArtificialLatencyInMs)
		} else {
			latencyResult, err := latency.GetLatency(target, 10)
			if err != nil {
				return localStores, remoteStores, fmt.Errorf("failed to get latency from %s", target)
			}
			storeLatency = latencyResult.Summary.Success.Average / 1000000
		}

		if storeLatency < c.RemoteStoreLatencyThresholdInMilliSec {
			if _, found := localStores[store.AvailabilityZone]; !found {
				locals := make([]consistent.RkvNode, 0)
				localStores[store.AvailabilityZone] = locals
			}
			localStores[store.AvailabilityZone] = append(localStores[store.AvailabilityZone],
				consistent.RkvNode{Name: target, Latency: time.Duration(storeLatency * int64(time.Millisecond)), IsRemote: false})
		} else {
			remoteStores = append(remoteStores, consistent.RkvNode{Name: target, Latency: time.Duration(storeLatency * int64(time.Millisecond)), IsRemote: true})
		}
	}
	fmt.Printf("The local stores are %v and the remote are %v", localStores, remoteStores)
	return localStores, remoteStores, nil
}

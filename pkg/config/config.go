package config

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path"
	"runtime"
	"strings"
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
	ConsistentHash string
	StoreType      string
	Stores         []KVStore
	BucketSize     int64
	ReplicaNum     int
	Concurrent     bool
}
type KVStore struct {
	RegionType            string
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

func (c *KVConfiguration) GetReplications() []string {
	var localStores []string
	var neighborStores []string
	var remoteStores []string
	// Please refer to https://user-images.githubusercontent.com/252020/184443299-799f1ed4-493a-4ea2-a9ed-72e15a2737af.png
	// for the following store categories.
	for _, store := range c.Stores {
		switch region := store.RegionType; region {
		case "local":
			localStores = append(localStores, store.Name)
		case "neighbor":
			neighborStores = append(neighborStores, store.Name)
		case "remote":
			remoteStores = append(remoteStores, store.Name)
		}
	}
	n := len(localStores)
	replications := make([]string, n)
	for idx := 0; idx < n; idx++ {
		total := c.ReplicaNum
		replics := make([]string, total)
		total--
		if remote := selectRandom(remoteStores); remote != "" {
			replics[total] = remote
			total--
		}
		if neighbor := selectRandom(neighborStores); neighbor != "" {
			replics[total] = neighbor
			total--
		}
		for i := 0; i < total+1; i++ {
			replics[total] = localStores[(idx+i)%n]
			total--
		}
		replications[idx] = strings.Join(replics, ",")
	}
	return replications
}

func selectRandom(array []string) string {
	if len(array) == 0 {
		return ""
	}
	randomIndex := rand.Intn(len(array))
	return array[randomIndex]
}

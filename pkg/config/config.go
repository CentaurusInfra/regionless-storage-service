package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
)

const (
	TraceName       = "regionless-kv-store"
	DefaultTraceEnv = "rkv-test"
)

var (
	TraceEnv string
)

type KVConfiguration struct {
	ConsistentHash string
	StoreType      string
	Stores         []KVStore
	BucketSize     int
}
type KVStore struct {
	Name string
	Host string
	Port int
}

func NewKVConfiguration(fileName string) (KVConfiguration, error) {
	_, runningfile, _, ok := runtime.Caller(1)
	configuration := KVConfiguration{}
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

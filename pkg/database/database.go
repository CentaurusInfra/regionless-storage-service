package database

import (
	"fmt"
	"time"

	"github.com/regionless-storage-service/pkg/config"
	"github.com/regionless-storage-service/pkg/constants"
	"github.com/regionless-storage-service/pkg/partition/consistent"
)

var (
	// Storages keeps all backend storages indexed by name
	Storages map[string]Database = make(map[string]Database)
)

type Database interface {
	Put(key, value string) (string, error)
	Get(key string) (string, error)
	Delete(key string) error
	Close() error
	Latency() time.Duration
	SetLatency(latency time.Duration)
}

func Factory(databaseType constants.StoreType, store *config.KVStore) (Database, error) {
	switch databaseType {
	case constants.Redis:
		databaseUrl := fmt.Sprintf("%s:%d", store.Host, store.Port)
		return createRedisDatabase(databaseUrl)
	case constants.Memory:
		databaseUrl := fmt.Sprintf("%s:%d", store.Host, store.Port)
		return NewMemDatabase(databaseUrl), nil
	case constants.DummyLatency: // simulator database backend suitable for internal perf load test
		return newLatencyDummyDatabase(time.Duration(store.ArtificialLatencyInMs) * time.Millisecond), nil
	default:
		return nil, &DatabaseNotImplementedError{databaseType.Name()}
	}
}

func FactoryByNode(databaseType constants.StoreType, store consistent.RkvNode) (Database, error) {
	switch databaseType {
	case constants.Redis:
		return createRedisDatabase(store.Name)
	case constants.Memory:
		return NewMemDatabase(store.Name), nil
	case constants.DummyLatency: // simulator database backend suitable for internal perf load test
		return newLatencyDummyDatabase(time.Duration(store.Latency) * time.Millisecond), nil
	default:
		return nil, &DatabaseNotImplementedError{databaseType.Name()}
	}
}

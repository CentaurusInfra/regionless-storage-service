package database

import (
	"fmt"
	"time"

	"github.com/regionless-storage-service/pkg/config"
	"github.com/regionless-storage-service/pkg/constants"
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

func FactoryWithNameAndLatency(databaseType constants.StoreType, name string, latency time.Duration) (Database, error) {
	switch databaseType {
	case constants.Redis:
		return createRedisDatabase(name)
	case constants.Memory:
		return NewMemDatabase(name), nil
	case constants.DummyLatency: // simulator database backend suitable for internal perf load test
		return newLatencyDummyDatabase(latency), nil
	default:
		return nil, &DatabaseNotImplementedError{databaseType.Name()}
	}
}

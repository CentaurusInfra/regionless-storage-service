package database

import (
	"fmt"
	"github.com/regionless-storage-service/pkg/config"
	"time"
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
}

func Factory(databaseType string, store *config.KVStore) (Database, error) {
	switch databaseType {
	case "redis":
		databaseUrl := fmt.Sprintf("%s:%d", store.Host, store.Port)
		return createRedisDatabase(databaseUrl)
	case "mem":
		databaseUrl := fmt.Sprintf("%s:%d", store.Host, store.Port)
		return NewMemDatabase(databaseUrl), nil
	case "dummy+latency": // simulator database backend suitable for internal perf load test
		return newLatencyDummyDatabase(time.Duration(store.ArtificialLatencyInMs) * time.Millisecond), nil
	default:
		return nil, &DatabaseNotImplementedError{databaseType}
	}
}

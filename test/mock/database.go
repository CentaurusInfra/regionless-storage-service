package mock

import (
	"errors"
	"time"

	"github.com/regionless-storage-service/pkg/database"
)

type MockDatabase struct {
	db           map[string]string
	readLatency  int
	writeLatency int
}

func NewMockDatabase() database.Database {
	return MockDatabase{db: make(map[string]string)}
}

func NewMockDatabaseWithLatency(readLatency, writeLatency int) database.Database {
	return MockDatabase{db: make(map[string]string), readLatency: readLatency, writeLatency: writeLatency}
}

func (md MockDatabase) Put(key, value string) (string, error) {
	if md.writeLatency > 0 {
		time.Sleep(time.Duration(md.writeLatency) * time.Second)
	}
	md.db[key] = value
	return "", nil
}

func (md MockDatabase) Get(key string) (string, error) {
	if md.readLatency > 0 {
		time.Sleep(time.Duration(md.readLatency) * time.Second)
	}
	if val, ok := md.db[key]; ok {
		return val, nil
	}
	return "", errors.New("key not found")
}

func (md MockDatabase) Delete(key string) error {
	delete(md.db, key)
	return nil
}

func (md MockDatabase) Close() error {
	return nil
}

func (md MockDatabase) Latency() time.Duration {
	return 0
}

func (md MockDatabase) SetLatency(latency time.Duration) {
}

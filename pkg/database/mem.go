package database

import (
	"errors"
	"time"
)

var memDatabases map[string]*MemDatabase

type MemDatabase struct {
	Name     string
	db       map[string]string
	wLatency int
}

func init() {
	memDatabases = make(map[string]*MemDatabase)
}
func NewMemDatabase(name string) Database {
	if md, ok := memDatabases[name]; ok {
		return md
	}
	md := MemDatabase{db: make(map[string]string), Name: name, wLatency: 1}
	memDatabases[name] = &md
	return md
}

func (md MemDatabase) Put(key, value string) (string, error) {
	if md.wLatency > 0 {
		time.Sleep(time.Duration(md.wLatency) * time.Second)
	}
	md.db[key] = value
	return "", nil
}

func (md MemDatabase) Get(key string) (string, error) {
	if val, ok := md.db[key]; ok {
		return val, nil
	}
	return "", errors.New("key not found")
}

func (md MemDatabase) Delete(key string) error {
	delete(md.db, key)
	return nil
}

func (md MemDatabase) Close() error {
	return nil
}

func (md MemDatabase) Latency() time.Duration {
	return 0
}

func (md MemDatabase) SetLatency(latency time.Duration) {
}

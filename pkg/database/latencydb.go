package database

import "time"

// latencyDatabase is decorator of adding specified latency to the underlying backend
type latencyDatabase struct {
	backend Database
	latency time.Duration
}

func (l latencyDatabase) Put(key, value string) (string, error) {
	time.Sleep(l.latency)
	return l.backend.Put(key, value)
}

func (l latencyDatabase) Get(key string) (string, error) {
	time.Sleep(l.latency)
	return l.backend.Get(key)
}

func (l latencyDatabase) Delete(key string) error {
	time.Sleep(l.latency)
	return l.backend.Delete(key)
}

func (l latencyDatabase) Close() error {
	// not to apply latency for close op
	return l.backend.Close()
}

// newLatencyDummyDatabase returns a simulated database backend which is able to apply fixed latency to CRUD ops
func newLatencyDummyDatabase(latency time.Duration) Database {
	return &latencyDatabase{backend: newDummyDatabase(), latency: latency}
}

package constants

import "time"

type StoreType string

func (r StoreType) Name() string {
	return string(r)
}

const (
	Memory       StoreType = "mem"
	Redis        StoreType = "redis"
	DummyLatency StoreType = "dummy+latency"
)

const (
	RedisRetryCount    int           = 5
	RedisRetryInterval time.Duration = 10 * time.Millisecond
)

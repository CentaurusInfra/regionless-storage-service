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
	RetryCount                  int           = 5
	RetryIntervalInMilliseconds time.Duration = 10 * time.Millisecond
)

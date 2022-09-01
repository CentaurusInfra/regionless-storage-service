package constants

type ConsistentHashingType string

func (r ConsistentHashingType) Name() string {
	return string(r)
}

const (
	Rendezvous ConsistentHashingType = "rendezvous"
	Ring       ConsistentHashingType = "ring"
)

type HashingManagerType string

func (hm HashingManagerType) Name() string {
	return string(hm)
}

const (
	SyncAsync HashingManagerType = "syncAsync"
	Sync      HashingManagerType = "sync"
)

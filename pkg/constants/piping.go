package constants

type PipingType string

func (p PipingType) Name() string {
	return string(p)
}

const (
	Chain                PipingType = "chain"
	LocalSyncRemoteAsync PipingType = "localSyncRemoteAsync"
)

package piping

import (
	"github.com/regionless-storage-service/pkg/consistent"
	"github.com/regionless-storage-service/pkg/consistent/chain"
	"github.com/regionless-storage-service/pkg/index"
)

type ChainPiping struct {
	databaseType string
	consistency  consistent.CONSISTENCY
}

func NewChainPiping(databaseType string, consistency consistent.CONSISTENCY) *ChainPiping {
	return &ChainPiping{databaseType: databaseType, consistency: consistency}
}

func (c *ChainPiping) Read(rev index.Revision) (string, error) {
	chain, err := chain.NewChain(c.databaseType, rev.GetNodes())
	if err != nil {
		return "", err
	}
	return chain.Read(rev.String(), c.consistency)
}

func (c *ChainPiping) Write(rev index.Revision, val string) error {
	chain, err := chain.NewChain(c.databaseType, rev.GetNodes())
	if err != nil {
		return err
	}
	chain.Write(rev.String(), val, c.consistency)
	return nil
}

package chain

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/regionless-storage-service/pkg/config"
	"github.com/regionless-storage-service/pkg/consistent"
	"github.com/regionless-storage-service/pkg/constants"
	"github.com/regionless-storage-service/pkg/database"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Chain struct {
	head, tail *ChainNode
	len        int
	ctx        context.Context
}

func NewChain(ctx context.Context, nodeType constants.StoreType, nodes []string) (*Chain, error) {
	n := len(nodes)
	if n == 0 {
		return nil, errors.New("the number of nodes is 0")
	}
	if n == 1 {
		return nil, errors.New("the number of nodes is 1, which means there is no replica")
	}
	dbs := make([]database.Database, n)
	if nodeType == constants.Memory {
		for i := 0; i < n; i++ {
			dbs[i] = database.NewMemDatabase(nodes[i])
		}
		return NewChainWithDatbases(ctx, dbs), nil
	}
	for i := 0; i < n; i++ {
		db, ok := database.Storages[nodes[i]]
		if ok {
			dbs[i] = db
		} else {
			return nil, fmt.Errorf("storage not found for %s", nodes[i])
		}
	}
	return NewChainWithDatbases(ctx, dbs), nil
}

func NewChainWithDatbases(ctx context.Context, dbs []database.Database) *Chain {
	dummy := NewNode(-1, nil)
	prev := dummy
	for i := 0; i < len(dbs); i++ {
		curr := NewNode(i, dbs[i])
		prev.next = curr
		prev = curr
	}
	return &Chain{head: dummy.next, tail: prev, len: len(dbs), ctx: ctx}
}

func (c *Chain) Write(key, val string, consistency consistent.CONSISTENCY) error {
	_, rootSpan := otel.Tracer(config.TraceName).Start(c.ctx, "db put")
	defer rootSpan.End()
	if _, err := c.head.db.Put(key, val); err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		return err
	}
	//Waiting for error handling design part
	if consistency == consistent.LINEARIZABLE {
		return c.head.next.Write(c.ctx, key, val)
	} else if consistency == consistent.SEQUENTIAL {
		go c.head.next.Write(c.ctx, key, val)
	}
	return nil
}

func (c *Chain) Delete(key string, consistency consistent.CONSISTENCY) error {
	_, rootSpan := otel.Tracer(config.TraceName).Start(c.ctx, "db delete")
	defer rootSpan.End()
	if err := c.head.db.Delete(key); err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		return err
	}
	//Waiting for error handling design part
	if consistency == consistent.LINEARIZABLE {
		return c.head.next.Delete(c.ctx, key)
	} else if consistency == consistent.SEQUENTIAL {
		go c.head.next.Delete(c.ctx, key)
	}
	return nil
}

func (c *Chain) Read(key string, consistency consistent.CONSISTENCY) (string, error) {
	if consistency == consistent.LINEARIZABLE {
		return c.tail.Read(c.ctx, key)
	} else if consistency == consistent.SEQUENTIAL {
		idx := rand.Intn(c.len)
		t := c.head
		for t != nil {
			if t.id == idx {
				return t.Read(c.ctx, key)
			}
			t = t.next
		}
	} else {
		return "", errors.New("consistency level does not implemented")
	}
	return "", errors.New("failed to read value")
}

func (c *Chain) GetHead() *ChainNode {
	return c.head
}

func (c *Chain) GetTail() *ChainNode {
	return c.tail
}

func (c *Chain) GetLen() int {
	return c.len
}

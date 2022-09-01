package piping

import (
	"context"
	"testing"

	"github.com/regionless-storage-service/pkg/consistent"
	"github.com/regionless-storage-service/pkg/index"
	"github.com/regionless-storage-service/pkg/piping"
)

func TestWriteLINEARIZABLE(t *testing.T) {
	cp := piping.NewChainPiping("mem", consistent.LINEARIZABLE, false)
	rev := index.NewRevision(1, 0, []string{"0.0.0.0:0", "1.1.1.1:1", "2.2.2.2:2", "3.3.3.3:3"})

	if err := cp.Write(context.TODO(), rev, "v"); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
}

func TestDeleteLINEARIZABLE(t *testing.T) {
	cp := piping.NewChainPiping("mem", consistent.LINEARIZABLE, false)
	rev := index.NewRevision(1, 0, []string{"0.0.0.0:0", "1.1.1.1:1", "2.2.2.2:2", "3.3.3.3:3"})
	if err := cp.Write(context.TODO(), rev, "v"); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
	if err := cp.Delete(context.TODO(), rev); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
}

func TestReadLINEARIZABLE(t *testing.T) {
	cp := piping.NewChainPiping("mem", consistent.LINEARIZABLE, false)
	rev := index.NewRevision(1, 0, []string{"0.0.0.0:0", "1.1.1.1:1", "2.2.2.2:2"})
	if err := cp.Write(context.TODO(), rev, "v"); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
	if val, err := cp.Read(context.TODO(), rev); err != nil {
		t.Fatalf("fail to read with the error %v", err)
	} else if val != "v" {
		t.Fatalf("read a wrong value %s", val)
	}
}

func TestWriteLINEARIZABLEConcurrently(t *testing.T) {
	cp := piping.NewChainPiping("mem", consistent.LINEARIZABLE, true)
	rev := index.NewRevision(1, 0, []string{"0.0.0.0:0", "1.1.1.1:1", "2.2.2.2:2", "3.3.3.3:3"})
	if err := cp.Write(context.TODO(), rev, "v"); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
}

func TestDeleteLINEARIZABLEConcurrently(t *testing.T) {
	cp := piping.NewChainPiping("mem", consistent.LINEARIZABLE, true)
	rev := index.NewRevision(1, 0, []string{"0.0.0.0:0", "1.1.1.1:1", "2.2.2.2:2", "3.3.3.3:3"})
	if err := cp.Write(context.TODO(), rev, "v"); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
	if err := cp.Delete(context.TODO(), rev); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
}

func TestReadLINEARIZABLEConcurrently(t *testing.T) {
	cp := piping.NewChainPiping("mem", consistent.LINEARIZABLE, true)
	rev := index.NewRevision(1, 0, []string{"0.0.0.0:0", "1.1.1.1:1", "2.2.2.2:2"})
	if err := cp.Write(context.TODO(), rev, "v"); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
	if val, err := cp.Read(context.TODO(), rev); err != nil {
		t.Fatalf("fail to read with the error %v", err)
	} else if val != "v" {
		t.Fatalf("read a wrong value %s", val)
	}
}

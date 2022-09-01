package piping

import (
	"context"
	"testing"

	"github.com/regionless-storage-service/pkg/constants"
	"github.com/regionless-storage-service/pkg/index"
	"github.com/regionless-storage-service/pkg/piping"
)

func TestWrite(t *testing.T) {
	sap := piping.NewSyncAsyncPiping(constants.Memory)
	rev := index.NewRevision(1, 0, []string{"1.1.1.1:80"})
	if err := sap.Write(context.TODO(), rev, "1"); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
}

func TestRead(t *testing.T) {
	sap := piping.NewSyncAsyncPiping(constants.Memory)
	rev := index.NewRevision(1, 0, []string{"1.1.1.1:80"})
	if err := sap.Write(context.TODO(), rev, "1"); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
	if v, err := sap.Read(context.TODO(), rev); err != nil {
		t.Fatalf("fail to read with the error %v", err)
	} else if v != "1" {
		t.Fatalf("The value shouldn't be %s", v)
	}
}

func TestDelete(t *testing.T) {
	sap := piping.NewSyncAsyncPiping(constants.Memory)
	rev := index.NewRevision(1, 0, []string{"1.1.1.1:80"})
	if err := sap.Write(context.TODO(), rev, "1"); err != nil {
		t.Fatalf("fail to write with the error %v", err)
	}
	if err := sap.Delete(context.TODO(), rev); err != nil {
		t.Fatalf("fail to delete  with the error %v", err)
	}
}

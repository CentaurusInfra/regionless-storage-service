package piping

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/regionless-storage-service/pkg/config"
	"github.com/regionless-storage-service/pkg/constants"
	"github.com/regionless-storage-service/pkg/database"
	"github.com/regionless-storage-service/pkg/index"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type SyncAsyncPiping struct {
	databaseType constants.StoreType
}

func NewSyncAsyncPiping(storeType constants.StoreType) *SyncAsyncPiping {
	return &SyncAsyncPiping{databaseType: storeType}
}

func (sap *SyncAsyncPiping) Read(ctx context.Context, rev index.Revision) (string, error) {
	_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "SyncAsyncPiping Read")
	defer rootSpan.End()
	syncNodes, _, err := splitStores(rev.GetNodes())
	if err != nil {
		return "", err
	}
	// The first sync store has the fewest latency. Threfore, it is chosen to read
	if database, err := database.FactoryWithNameAndLatency(sap.databaseType, syncNodes[0], 0); err != nil {
		return "", err
	} else {
		return database.Get(rev.String())
	}
}

func (sap *SyncAsyncPiping) Write(ctx context.Context, rev index.Revision, val string) error {
	_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "SyncAsyncPiping write")
	defer rootSpan.End()
	syncNodes, asyncNodes, err := splitStores(rev.GetNodes())
	if err != nil {
		return err
	}

	for _, asyncNode := range asyncNodes {
		go func(ctx context.Context, databaseType constants.StoreType, name, key, val string) {
			_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "async db put")
			defer rootSpan.End()
			if database, err := database.FactoryWithNameAndLatency(sap.databaseType, name, 0); err != nil {
				rootSpan.RecordError(err)
				rootSpan.SetStatus(codes.Error, err.Error())
			} else {
				if _, err := database.Put(key, val); err != nil {
					rootSpan.RecordError(err)
					rootSpan.SetStatus(codes.Error, err.Error())
				}
			}

		}(ctx, sap.databaseType, asyncNode, rev.String(), val)
	}

	var wg sync.WaitGroup
	for _, syncNode := range syncNodes {
		wg.Add(1)
		go func(ctx context.Context, databaseType constants.StoreType, name, key, val string) {
			defer wg.Done()
			_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "sync db put")
			defer rootSpan.End()
			if database, err := database.FactoryWithNameAndLatency(sap.databaseType, name, 0); err != nil {
				rootSpan.RecordError(err)
				rootSpan.SetStatus(codes.Error, err.Error())
			} else {
				if _, err := database.Put(key, val); err != nil {
					rootSpan.RecordError(err)
					rootSpan.SetStatus(codes.Error, err.Error())
				}
			}

		}(ctx, sap.databaseType, syncNode, rev.String(), val)
	}
	wg.Wait()

	return nil
}

func (sap *SyncAsyncPiping) Delete(ctx context.Context, rev index.Revision) error {
	_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "SyncAsyncPiping delete")
	defer rootSpan.End()
	syncNodes, asyncNodes, err := splitStores(rev.GetNodes())
	if err != nil {
		return err
	}

	for _, asyncNode := range asyncNodes {
		go func(ctx context.Context, databaseType constants.StoreType, name, key string) {
			_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "async db delete")
			defer rootSpan.End()
			if database, err := database.FactoryWithNameAndLatency(sap.databaseType, name, 0); err != nil {
				rootSpan.RecordError(err)
				rootSpan.SetStatus(codes.Error, err.Error())
			} else {
				if err := database.Delete(key); err != nil {
					rootSpan.RecordError(err)
					rootSpan.SetStatus(codes.Error, err.Error())
				}
			}

		}(ctx, sap.databaseType, asyncNode, rev.String())
	}

	var wg sync.WaitGroup
	for _, syncNode := range syncNodes {
		wg.Add(1)
		go func(ctx context.Context, databaseType constants.StoreType, name, key string) {
			defer wg.Done()
			_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "sync db delete")
			defer rootSpan.End()
			if database, err := database.FactoryWithNameAndLatency(sap.databaseType, name, 0); err != nil {
				rootSpan.RecordError(err)
				rootSpan.SetStatus(codes.Error, err.Error())
			} else {
				if err := database.Delete(key); err != nil {
					rootSpan.RecordError(err)
					rootSpan.SetStatus(codes.Error, err.Error())
				}
			}

		}(ctx, sap.databaseType, syncNode, rev.String())
	}
	wg.Wait()

	return nil
}

func splitStores(stores []string) ([]string, []string, error) {
	syncNodes := make([]string, 0)
	asyncNodes := make([]string, 0)
	if len(stores) < 1 {
		return syncNodes, asyncNodes, fmt.Errorf("no stores in the revision")
	}
	syncNodes = strings.Split(stores[0], ",")
	if len(syncNodes) < 1 {
		return syncNodes, asyncNodes, fmt.Errorf("no sync stores in the revision")
	}
	if len(stores) == 2 {
		asyncNodes = strings.Split(stores[1], ",")
	}
	return syncNodes, asyncNodes, nil
}

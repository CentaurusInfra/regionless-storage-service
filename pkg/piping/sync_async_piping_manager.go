package piping

import (
	"context"
	"sort"
	"sync"

	"github.com/regionless-storage-service/pkg/config"
	"github.com/regionless-storage-service/pkg/constants"
	"github.com/regionless-storage-service/pkg/database"
	"github.com/regionless-storage-service/pkg/index"
	"github.com/regionless-storage-service/pkg/partition/consistent"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type SyncAsyncPiping struct {
	databaseType constants.StoreType
	localStores  []database.Database
	remoteStores []database.Database
}

func NewSyncAsyncPiping(storeType constants.StoreType, nodes []consistent.RkvNode) (*SyncAsyncPiping, error) {
	localStores := make([]database.Database, 0)
	remoteStores := make([]database.Database, 0)
	for _, node := range nodes {
		if node.IsRemote {
			if remoteStore, err := database.FactoryByNode(storeType, node); err != nil {
				return nil, err
			} else {
				remoteStores = append(remoteStores, remoteStore)
			}
		} else {
			if localStore, err := database.FactoryByNode(storeType, node); err != nil {
				return nil, err
			} else {
				localStores = append(localStores, localStore)
			}
		}
	}

	sort.Slice(localStores, func(i, j int) bool {
		return localStores[i].Latency() < localStores[j].Latency()
	})

	return &SyncAsyncPiping{databaseType: storeType, localStores: localStores, remoteStores: remoteStores}, nil
}

func (sap *SyncAsyncPiping) Read(ctx context.Context, rev index.Revision) (string, error) {
	_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "SyncAsyncPiping Read")
	defer rootSpan.End()

	return sap.localStores[0].Get(rev.String())
}

func (sap *SyncAsyncPiping) Write(ctx context.Context, rev index.Revision, val string) error {
	_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "SyncAsyncPiping write")
	defer rootSpan.End()

	for _, remoteStore := range sap.remoteStores {
		go func(ctx context.Context, store database.Database, key, val string) {
			_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "remote db put")
			defer rootSpan.End()
			if _, err := store.Put(key, val); err != nil {
				rootSpan.RecordError(err)
				rootSpan.SetStatus(codes.Error, err.Error())
			}
		}(ctx, remoteStore, rev.String(), val)
	}

	var wg sync.WaitGroup
	for _, localStore := range sap.localStores {
		wg.Add(1)
		go func(ctx context.Context, store database.Database, key, val string) {
			defer wg.Done()
			_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "local db put")
			defer rootSpan.End()
			if _, err := store.Put(key, val); err != nil {
				rootSpan.RecordError(err)
				rootSpan.SetStatus(codes.Error, err.Error())
			}
		}(ctx, localStore, rev.String(), val)
	}
	wg.Wait()

	return nil
}

func (sap *SyncAsyncPiping) Delete(ctx context.Context, rev index.Revision) error {
	_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "SyncAsyncPiping delete")
	defer rootSpan.End()

	for _, remoteStore := range sap.remoteStores {
		go func(ctx context.Context, store database.Database, key string) {
			_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "remote db delete")
			defer rootSpan.End()
			if err := store.Delete(key); err != nil {
				rootSpan.RecordError(err)
				rootSpan.SetStatus(codes.Error, err.Error())
			}
		}(ctx, remoteStore, rev.String())
	}

	var wg sync.WaitGroup
	for _, localStore := range sap.localStores {
		wg.Add(1)
		go func(ctx context.Context, store database.Database, key string) {
			defer wg.Done()
			_, rootSpan := otel.Tracer(config.TraceName).Start(ctx, "local db delete")
			defer rootSpan.End()
			if err := store.Delete(key); err != nil {
				rootSpan.RecordError(err)
				rootSpan.SetStatus(codes.Error, err.Error())
			}
		}(ctx, localStore, rev.String())
	}
	wg.Wait()

	return nil
}

func splitLocalAndRemoteStores(stores []consistent.RkvNode) ([]consistent.RkvNode, []consistent.RkvNode) {
	localStores := make([]consistent.RkvNode, 0)
	remoteStores := make([]consistent.RkvNode, 0)
	for _, store := range stores {
		if store.IsRemote {
			remoteStores = append(remoteStores, store)
		} else {
			localStores = append(localStores, store)
		}
	}
	return localStores, remoteStores
}

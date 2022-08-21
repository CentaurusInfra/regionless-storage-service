package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/regionless-storage-service/pkg/config"
	ca "github.com/regionless-storage-service/pkg/consistent"
	"github.com/regionless-storage-service/pkg/index"
	"github.com/regionless-storage-service/pkg/partition/consistent"
	"github.com/regionless-storage-service/pkg/piping"
	"github.com/regionless-storage-service/pkg/revision"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"k8s.io/klog"
	"net/http"
	"strconv"
	"strings"
)

type KeyValueHandler struct {
	ch        consistent.ConsistentHashing
	conf      config.KVConfiguration
	indexTree index.Index
	piping    piping.Piping
}

func (handler *KeyValueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/kv" {
		http.NotFound(w, r)
		return
	}
	var result string
	var statusCode int
	var err error

	switch r.Method {
	case "GET":
		result, err = handler.getKV(w, r)
		statusCode = http.StatusAccepted
	case "POST":
		result, err = handler.createKV(w, r)
		statusCode = http.StatusCreated
	case "PUT":
		result, err = handler.createKV(w, r)
		statusCode = http.StatusCreated
	case "DELETE":
		result, err = handler.deleteKV(w, r)
		statusCode = http.StatusAccepted
	default:
		result = http.StatusText(http.StatusNotImplemented)
		statusCode = http.StatusNotImplemented
	}
	if err != nil {
		internalError := http.StatusInternalServerError
		http.Error(w, err.Error(), internalError)
	} else if result != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(result))
	}
}

func (handler *KeyValueHandler) getKV(w http.ResponseWriter, r *http.Request) (string, error) {
	// tracing getkv op
	ctx, span := otel.Tracer(config.TraceName).Start(r.Context(), "getKV")
	defer span.End()

	key, ok := r.URL.Query()["key"]
	if ok {
		fromRevs, hasRev := r.URL.Query()["fromRev"]
		if hasRev {
			fromRev, err := strconv.Atoi(fromRevs[0])
			if err != nil {
				return "", err
			}

			revs := handler.indexTree.RangeSince(ctx, []byte(key[0]), nil, int64(fromRev))

			{
				_, span := otel.Tracer(config.TraceName).Start(ctx, "get kv", trace.WithSpanKind(trace.SpanKindClient))
				defer span.End()
				rets, err := handler.getValuesByRevs(ctx, revs)
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return "", err
				}
				m := make(map[string]string, 0)
				for i := 0; i < len(revs); i++ {
					m[revs[i].String()] = rets[i]
				}
				return fmt.Sprintf("The values are %s from the revision %d\n", fmt.Sprint(m), fromRev), nil
			}
		} else {
			rev, _, _, err := handler.indexTree.Get(ctx, []byte(key[0]), 0)

			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return "", err
			}

			{
				_, span := otel.Tracer(config.TraceName).Start(ctx, "get kv", trace.WithSpanKind(trace.SpanKindClient))
				defer span.End()
				ret, err := handler.getValueByRev(ctx, rev)
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return "", err
				}

				return fmt.Sprintf("The value is %s with the revision %s\n", ret, rev.String()), nil
			}
		}
	}
	return "", fmt.Errorf("the key is missing at the query %v", r.URL.Query())
}

func (handler *KeyValueHandler) getValueByRev(ctx context.Context, rev index.Revision) (string, error) {
	ctx, span := otel.Tracer(config.TraceName).Start(ctx, "getValueByRev")
	defer span.End()
	ret, err := handler.piping.Read(ctx, rev)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	return ret, nil
}

func (handler *KeyValueHandler) getValuesByRevs(ctx context.Context, revs []index.Revision) ([]string, error) {
	ctx, span := otel.Tracer(config.TraceName).Start(ctx, "getValuesByRevs")
	defer span.End()
	n := len(revs)
	res := make([]string, n)
	for i := 0; i < n; i++ {
		if ret, err := handler.getValueByRev(ctx, revs[i]); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		} else {
			res[i] = string(ret)
		}
	}
	return res, nil
}

func (handler *KeyValueHandler) createKV(w http.ResponseWriter, r *http.Request) (string, error) {
	// tracing createkv op
	ctx, rootSpan := otel.Tracer(config.TraceName).Start(r.Context(), "createKV")
	defer rootSpan.End()

	rev := revision.GetGlobalIncreasingRevision()
	newRev := index.NewRevision(int64(rev), 0, nil)
	node := handler.ch.LocateKey(handler.getPrimaryRevBytesWithBucket(newRev))
	newRev.SetNodes(strings.Split(node.String(), ","))

	byteValue, err := ioutil.ReadAll(r.Body)

	if err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		klog.Errorf("Failed to read key value with the error %v", err)
		return "", err
	}
	payload := map[string]string{}
	err = json.Unmarshal(byteValue, &payload)

	if err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		return "", err
	}

	{
		_, span := otel.Tracer(config.TraceName).Start(ctx, "set kv", trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()
		err := handler.piping.Write(ctx, newRev, payload["value"])
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}

	if err = handler.indexTree.Put(ctx, []byte(payload["key"]), newRev); err != nil {
		// todo: cleanup writes on nodes
		return "", err
	}

	return fmt.Sprintf("The key value pair (%s,%s) has been saved as revision %s at %s\n", payload["key"], payload["value"], strconv.FormatUint(rev, 10), node.String()), err
}

func (handler *KeyValueHandler) deleteKV(w http.ResponseWriter, r *http.Request) (string, error) {
	// tracing deletekv op
	ctx, rootSpan := otel.Tracer(config.TraceName).Start(r.Context(), "deleteKV")
	defer rootSpan.End()

	key, ok := r.URL.Query()["key"]
	if ok {
		rev, _, _, err := handler.indexTree.Get(ctx, []byte(key[0]), 0)
		if err != nil {
			rootSpan.RecordError(err)
			rootSpan.SetStatus(codes.Error, err.Error())
			return "", err
		}

		{
			_, span := otel.Tracer(config.TraceName).Start(ctx, "delete kv", trace.WithSpanKind(trace.SpanKindClient))
			defer span.End()
			err = handler.piping.Delete(ctx, rev)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
		}

		handler.indexTree.Tombstone(ctx, []byte(key[0]), index.NewRevision(int64(revision.GetGlobalIncreasingRevision()), rev.GetSub(), nil))

		return fmt.Sprintf("The key %s has been removed at %s\n", key, rev.GetNodes()), err
	}
	return "", fmt.Errorf("the key is missing at the query %v", r.URL.Query())
}

func (handler *KeyValueHandler) getPrimaryRevBytesWithBucket(rev index.Revision) []byte {
	primaryRev := rev.GetMain() / handler.conf.BucketSize
	primaryRevBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(primaryRevBytes, uint64(primaryRev))
	return primaryRevBytes
}

func NewKeyValueHandler(conf config.KVConfiguration) *KeyValueHandler {
	ring := consistent.NewRendezvous(nil, rkvHash{})
	stores := conf.GetReplications()
	for _, store := range stores {
		ring.AddNode(rkvNode(store))
	}
	piping := piping.NewChainPiping(conf.StoreType, ca.LINEARIZABLE, conf.Concurrent)
	return &KeyValueHandler{ch: ring, conf: conf, indexTree: index.NewTreeIndex(), piping: piping}
}

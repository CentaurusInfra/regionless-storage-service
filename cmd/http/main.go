package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/cespare/xxhash"
	"k8s.io/klog"

	"github.com/regionless-storage-service/pkg/config"
	"github.com/regionless-storage-service/pkg/database"
	"github.com/regionless-storage-service/pkg/index"
	"github.com/regionless-storage-service/pkg/partition/consistent"
	"github.com/regionless-storage-service/pkg/revision"
)

func main() {
	// -trace-env="onebox-730", for instance, is a good name for 730 milestone, one-box rkv system
	flag.StringVar(&config.TraceEnv, "trace-env", config.DefaultTraceEnv, "environment name displayed in tracing system")
	jaegerServer := flag.String("jaeger-server", "http://localhost:14268", "jaeger server endpoint in form of http://host-ip:port")
	flag.Parse()

	// for now, only support http protocol of jaeger service
	jaegerEndpoint := *jaegerServer + "/api/traces"

	traceProvider, err := tracerProvider(jaegerEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	otel.SetTracerProvider(traceProvider)

	conf, err := config.NewKVConfiguration("config.json")
	if err != nil {
		panic(fmt.Errorf("error setting gateway agent configuration: %v", err))
	}
	database.InitPool(conf.Stores)

	url := flag.String("url", ":8090", "proxy url")
	flag.Parse()
	keyValueHandler := NewKeyValueHandler(conf)
	if keyValueHandler == nil {
		klog.Error("cannot run http server - http handler is null")
	} else {
		http.Handle("/kv", keyValueHandler)
		klog.Fatal(http.ListenAndServe(*url, nil))
	}
}

type KeyValueHandler struct {
	mu        sync.Mutex
	ch        consistent.ConsistentHashing
	conf      config.KVConfiguration
	indexTree index.Index
}
type testNode string

func (tn testNode) String() string {
	return string(tn)
}

type testHash struct{}

func (th testHash) Hash(key []byte) uint64 {
	return xxhash.Sum64(key)
}

type KV struct {
	key, value string
}

func NewKeyValueHandler(conf config.KVConfiguration) *KeyValueHandler {
	ring := consistent.NewRendezvous(nil, testHash{})
	for _, store := range conf.Stores {
		node := fmt.Sprintf("%s:%d", store.Host, store.Port)
		ring.AddNode(testNode(node))
	}
	return &KeyValueHandler{ch: ring, conf: conf, indexTree: index.NewTreeIndex()}
}

func (handler *KeyValueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler.mu.Lock()
	defer handler.mu.Unlock()
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

			{
				_, span := otel.Tracer(config.TraceName).Start(ctx, "rangesince kv", trace.WithSpanKind(trace.SpanKindClient))
				defer span.End()

				revs := handler.indexTree.RangeSince([]byte(key[0]), nil, int64(fromRev))
				rets, err := handler.getValuesByRevs(revs)
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
				ret, err := handler.getValueByRev(rev)
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
func (handler *KeyValueHandler) getValueByRev(rev index.Revision) (string, error) {
	node := handler.ch.LocateKey([]byte(rev.String()))
	conn, err := database.Factory(handler.conf.StoreType, node.String())
	if err != nil {
		return "", err
	}
	ret, err := conn.Get(rev.String())
	if err != nil {
		return "", err
	}
	return ret, nil
}

func (handler *KeyValueHandler) getValuesByRevs(revs []index.Revision) ([]string, error) {
	n := len(revs)
	res := make([]string, n)
	for i := 0; i < n; i++ {
		if ret, err := handler.getValueByRev(revs[i]); err != nil {
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
	node := handler.ch.LocateKey([]byte(strconv.FormatUint(rev, 10)))
	conn, err := database.Factory(handler.conf.StoreType, node.String())
	if err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		return "", err
	}

	byteValue, err := ioutil.ReadAll(r.Body)

	if err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		klog.Errorf("Failed to read key value with the error %v", err)
		return "", err
	}
	x := map[string]string{}
	err = json.Unmarshal(byteValue, &x)

	if err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, err.Error())
		return "", err
	}

	{
		_, span := otel.Tracer(config.TraceName).Start(ctx, "set kv", trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()
		_, err = conn.Put(strconv.FormatUint(rev, 10), x["value"])
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}

	{
		_, span := otel.Tracer(config.TraceName).Start(ctx, "put index")
		defer span.End()
		handler.indexTree.Put([]byte(x["key"]), index.NewRevision(int64(rev), 0))
	}

	return fmt.Sprintf("The key value pair (%s,%s) has been saved as revision %s at %s\n", x["key"], x["value"], strconv.FormatUint(rev, 10), node.String()), err
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
		node := handler.ch.LocateKey([]byte(rev.String()))
		conn, err := database.Factory(handler.conf.StoreType, node.String())
		if err != nil {
			rootSpan.RecordError(err)
			rootSpan.SetStatus(codes.Error, err.Error())
			return "", err
		}

		{
			_, span := otel.Tracer(config.TraceName).Start(ctx, "delete kv", trace.WithSpanKind(trace.SpanKindClient))
			defer span.End()
			err = conn.Delete(rev.String())
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
		}

		{
			_, span := otel.Tracer(config.TraceName).Start(ctx, "tombstone index")
			defer span.End()
			handler.indexTree.Tombstone([]byte(key[0]), rev)
		}

		return fmt.Sprintf("The key %s has been removed at %s\n", key, node.String()), err
	}
	return "", fmt.Errorf("the key is missing at the query %v", r.URL.Query())
}

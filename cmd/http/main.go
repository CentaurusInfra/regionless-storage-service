package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/cespare/xxhash"
	"k8s.io/klog"

	"github.com/regionless-storage-service/pkg/config"
	"github.com/regionless-storage-service/pkg/database"
	"github.com/regionless-storage-service/pkg/tracer"
)

func main() {
	// For now, we use the current time as seed for each configuration. However, we might notice that
	// it will give a deterministic sequence of pseudo-random numbers as the code shows according to
	// its implementation https://github.com/golang/go/blob/master/src/math/rand/rng.go#L25
	rand.Seed(time.Now().UnixNano())

	// -trace-env="onebox-730", for instance, is a good name for 730 milestone, one-box rkv system
	flag.StringVar(&config.TraceEnv, "trace-env", config.DefaultTraceEnv, "environment name displayed in tracing system")
	jaegerServer := flag.String("jaeger-server", "http://localhost:14268", "jaeger server endpoint in form of http://host-ip:port")
	flag.Float64Var(&config.TraceSamplingRate, "trace-sampling-rate", 1.0, "optional sampling rate")
	url := flag.String("url", ":8090", "proxy url")
	flag.Parse()
	if len(config.TraceEnv) != 0 {
		tracer.SetupTracer(jaegerServer)
	}

	conf, err := config.NewKVConfiguration("config.json")
	if err != nil {
		panic(fmt.Errorf("error setting gateway agent configuration: %v", err))
	}
	database.InitStorageInstancePool(conf.Stores)

	http.Handle("/kv", NewKeyValueHandler(conf))
	klog.Fatal(http.ListenAndServe(*url, nil))
}

type rkvNode string

func (tn rkvNode) String() string {
	return string(tn)
}

type rkvHash struct{}

func (th rkvHash) Hash(key []byte) uint64 {
	return xxhash.Sum64(key)
}

type KV struct {
	key, value string
}

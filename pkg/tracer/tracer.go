package tracer

import (
	"github.com/regionless-storage-service/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"log"
	"math"
	"math/rand"
)

func SetupTracer(jaegerServer *string) {
	// for now, only support http protocol of jaeger service
	jaegerEndpoint := *jaegerServer + "/api/traces"
	traceProvider, err := newTracerProvider(jaegerEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTracerProvider(traceProvider)
}

func newTracerProvider(url string) (*trace.TracerProvider, error) {
	// default sampler is the always-on
	var sampler trace.Sampler
	if config.TraceSamplingRate < 1.0 { // use the custom probability sampler to reduce the trace samples
		sampler = trace.ParentBased(newProbabilisticSampler(config.TraceSamplingRate))
	}

	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(
		// Always be sure to batch in production.
		trace.WithBatcher(exp),
		// Record information about this application in an Resource.
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.TraceName),
			attribute.String("environment", config.TraceEnv),
		)),
		trace.WithSampler(sampler),
	)
	return tp, nil
}

type probabilisticSampler struct {
	max      int
	boundary int // values equal or greater than boundary would be dropped; 1/00 prob can be expressed as max=100, boundary=1
}

func (s probabilisticSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	r := rand.Intn(s.max)
	if r >= s.boundary {
		return trace.SamplingResult{
			Decision:   trace.Drop,
			Tracestate: oteltrace.SpanContextFromContext(p.ParentContext).TraceState(),
		}
	}

	return trace.SamplingResult{
		Decision:   trace.RecordAndSample,
		Tracestate: oteltrace.SpanContextFromContext(p.ParentContext).TraceState(),
	}
}

func (s probabilisticSampler) Description() string {
	return "probabilistic sampler"
}

func newProbabilisticSampler(probability float64) *probabilisticSampler {
	boundary := math.MaxInt32 * probability
	return &probabilisticSampler{math.MaxInt32, int(boundary)}
}

var _ trace.Sampler = newProbabilisticSampler(0.001)

package main

import (
	"github.com/regionless-storage-service/pkg/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"math"
	"math/rand"
)

func tracerProvider(url string) (*trace.TracerProvider, error) {
	// default sampler is the always-on
	var sampler trace.Sampler
	if config.TraceSamplingRate < 1.0 { // use the custom probability sampler to reduce the trace samples
		sampler = trace.ParentBased(NewProbabilisticSampler(config.TraceSamplingRate))
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

type ProbabilisticSampler struct {
	max      int
	boundary int // values equal or greater than boundary would be dropped; 1/00 prob can be expressed as max=100, boundary=1
}

func (s ProbabilisticSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
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

func (s ProbabilisticSampler) Description() string {
	return "probabilistic sampler"
}

func NewProbabilisticSampler(probability float64) *ProbabilisticSampler {
	boundary := math.MaxInt32 * probability
	return &ProbabilisticSampler{math.MaxInt32, int(boundary)}
}

var _ trace.Sampler = NewProbabilisticSampler(0.001)

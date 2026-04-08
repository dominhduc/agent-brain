package otel

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracerProvider trace.TracerProvider
	globalShutdown func(context.Context) error
	version        = "v0.20.0"
)

func Init(ctx context.Context, cfg Config) error {
	if !cfg.Enabled {
		tracerProvider = trace.NewNoopTracerProvider()
		return nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("brain"),
			semconv.ServiceVersion(version),
			attribute.String("brain.project", getProjectName()),
		),
	)
	if err != nil {
		return fmt.Errorf("creating OTel resource: %w", err)
	}

	var exporter sdktrace.SpanExporter

	switch strings.ToLower(cfg.Endpoint) {
	case "stdout":
		exporter = &stdoutExporter{}
	case "none", "":
		tracerProvider = trace.NewNoopTracerProvider()
		return nil
	default:
		if strings.HasPrefix(cfg.Endpoint, "http://") || strings.HasPrefix(cfg.Endpoint, "https://") {
			opts := []otlptracehttp.Option{
				otlptracehttp.WithEndpointURL(cfg.Endpoint),
			}
			for k, v := range cfg.Headers {
				opts = append(opts, otlptracehttp.WithHeaders(map[string]string{k: v}))
			}
			exporter, err = otlptracehttp.New(ctx, opts...)
		} else {
			opts := []otlptracegrpc.Option{
				otlptracegrpc.WithEndpoint(cfg.Endpoint),
				otlptracegrpc.WithInsecure(),
			}
			for k, v := range cfg.Headers {
				opts = append(opts, otlptracegrpc.WithHeaders(map[string]string{k: v}))
			}
			exporter, err = otlptracegrpc.New(ctx, opts...)
		}
		if err != nil {
			return fmt.Errorf("creating OTLP exporter: %w", err)
		}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracerProvider = tp
	globalShutdown = tp.Shutdown

	return nil
}

func Shutdown(ctx context.Context) error {
	if globalShutdown != nil {
		return globalShutdown(ctx)
	}
	return nil
}

func Tracer() trace.Tracer {
	if tracerProvider == nil {
		return trace.NewNoopTracerProvider().Tracer("brain")
	}
	return tracerProvider.Tracer("brain")
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

func RecordEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

func SetAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}
	span.SetAttributes(attrs...)
}

func EndSpan(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(BrainOutcome.String("error"))
	} else {
		span.SetAttributes(BrainOutcome.String("success"))
	}
	span.End()
}

func getProjectName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	parts := strings.Split(cwd, string(os.PathSeparator))
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

type stdoutExporter struct{}

func (e *stdoutExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, s := range spans {
		attrs := make(map[string]interface{})
		for _, a := range s.Attributes() {
			attrs[string(a.Key)] = a.Value.AsInterface()
		}
		fmt.Fprintf(os.Stderr,
			"[%s] span: %s trace_id=%s span_id=%s duration=%s attrs=%v\n",
			time.Now().Format(time.RFC3339),
			s.Name(),
			s.SpanContext().TraceID().String(),
			s.SpanContext().SpanID().String(),
			s.EndTime().Sub(s.StartTime()),
			attrs,
		)
	}
	return nil
}

func (e *stdoutExporter) Shutdown(ctx context.Context) error {
	return nil
}

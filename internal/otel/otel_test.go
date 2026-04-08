package otel

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func resetTracerProvider() {
	tracerProvider = nil
	globalShutdown = nil
}

func TestInitDisabledReturnsNoopTracer(t *testing.T) {
	defer resetTracerProvider()

	cfg := Config{Enabled: false, Endpoint: ""}
	err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	_, span := StartSpan(context.Background(), "test-span")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()

	tr := Tracer()
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestInitStdoutEndpoint(t *testing.T) {
	defer resetTracerProvider()

	cfg := Config{Enabled: true, Endpoint: "stdout"}
	err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	_, span := StartSpan(context.Background(), "stdout-test-span")
	span.SetAttributes(attribute.String("test.key", "test.value"))
	span.End()

	err = Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestInitEmptyEndpointReturnsNoop(t *testing.T) {
	defer resetTracerProvider()

	cfg := Config{Enabled: true, Endpoint: ""}
	err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	_, span := StartSpan(context.Background(), "test-span")
	span.End()
}

type testExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func (e *testExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *testExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *testExporter) GetSpans() []sdktrace.ReadOnlySpan {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]sdktrace.ReadOnlySpan, len(e.spans))
	copy(result, e.spans)
	return result
}

func TestStartSpanEndSpan(t *testing.T) {
	defer resetTracerProvider()

	exporter := &testExporter{}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("brain-test"),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	tracerProvider = tp
	globalShutdown = tp.Shutdown

	_, span := StartSpan(context.Background(), "my-operation")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if span.SpanContext().TraceID().IsValid() != true {
		t.Fatal("expected valid trace ID")
	}

	time.Sleep(5 * time.Millisecond)

	EndSpan(span, nil)

	tp.ForceFlush(context.Background())

	exported := exporter.GetSpans()
	if len(exported) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(exported))
	}

	if exported[0].Name() != "my-operation" {
		t.Errorf("expected span name 'my-operation', got %q", exported[0].Name())
	}

	found := false
	for _, attr := range exported[0].Attributes() {
		if string(attr.Key) == "brain.outcome" && attr.Value.AsString() == "success" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected brain.outcome=success attribute")
	}
}

func TestAttributesSetCorrectly(t *testing.T) {
	defer resetTracerProvider()

	exporter := &testExporter{}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("brain-test"),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	tracerProvider = tp
	globalShutdown = tp.Shutdown

	_, span := StartSpan(context.Background(), "attr-test")
	SetAttributes(span, BrainCommand.String("test-cmd"), BrainLLMModel.String("gpt-4"))
	RecordEvent(span, "test-event", attribute.Int("test.count", 42))
	EndSpan(span, nil)

	tp.ForceFlush(context.Background())

	exported := exporter.GetSpans()
	if len(exported) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(exported))
	}

	attrMap := make(map[string]interface{})
	for _, a := range exported[0].Attributes() {
		attrMap[string(a.Key)] = a.Value.AsInterface()
	}

	if v, ok := attrMap["brain.command"]; !ok || v != "test-cmd" {
		t.Errorf("expected brain.command=test-cmd, got %v", attrMap["brain.command"])
	}
	if v, ok := attrMap["brain.llm.model"]; !ok || v != "gpt-4" {
		t.Errorf("expected brain.llm.model=gpt-4, got %v", attrMap["brain.llm.model"])
	}

	events := exported[0].Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Name != "test-event" {
		t.Errorf("expected event name 'test-event', got %q", events[0].Name)
	}

	found := false
	for _, a := range events[0].Attributes {
		if string(a.Key) == "test.count" && a.Value.AsInt64() == 42 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected test.count=42 in event attributes")
	}
}

func TestEndSpanWithError(t *testing.T) {
	defer resetTracerProvider()

	exporter := &testExporter{}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("brain-test"),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	tracerProvider = tp
	globalShutdown = tp.Shutdown

	testErr := errors.New("something went wrong")
	_, span := StartSpan(context.Background(), "error-test")
	EndSpan(span, testErr)

	tp.ForceFlush(context.Background())

	exported := exporter.GetSpans()
	if len(exported) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(exported))
	}

	found := false
	for _, attr := range exported[0].Attributes() {
		if string(attr.Key) == "brain.outcome" && attr.Value.AsString() == "error" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected brain.outcome=error attribute")
	}

	status := exported[0].Status()
	if status.Code != 0 {
		t.Errorf("expected unset status code (0), got %d", status.Code)
	}

	eventFound := false
	for _, ev := range exported[0].Events() {
		if ev.Name == "exception" {
			eventFound = true
			break
		}
	}
	if !eventFound {
		t.Error("expected exception event from RecordError")
	}
}

func TestRecordEventWithNilSpan(t *testing.T) {
	RecordEvent(nil, "event", attribute.String("k", "v"))
}

func TestSetAttributesWithNilSpan(t *testing.T) {
	SetAttributes(nil, attribute.String("k", "v"))
}

func TestEndSpanWithNilSpan(t *testing.T) {
	EndSpan(nil, nil)
}

func TestTracerReturnsValidTracer(t *testing.T) {
	defer resetTracerProvider()

	tr := Tracer()
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}

	_, span := tr.Start(context.Background(), "direct-tracer-test")
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
}

func TestShutdownWithoutInit(t *testing.T) {
	defer resetTracerProvider()
	err := Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown should be noop when not initialized, got error: %v", err)
	}
}

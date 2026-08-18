package tracing

import (
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// init registers a local TracerProvider so every span gets a real randomly generated trace/span ID - even with no exporter configured.
// these IDs are valid and usable to correlate our own log lines

func Init(){
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
}

// we're only using openTelemetry's ID generator to get real trace/span IDs into our logs.
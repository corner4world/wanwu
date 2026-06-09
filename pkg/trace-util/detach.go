package trace_util

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// DetachContext extracts trace propagation info from src and injects it into
// a new context.Background(). The returned context is NOT cancelled when src is
// cancelled, but preserves the trace chain (traceID, spanID) so downstream
// calls (e.g. gRPC) can propagate the trace correctly.
//
// Typical use-case: goroutines that must outlive the HTTP request lifecycle
// (e.g. async statistics recording) but still need trace correlation.
func DetachContext(src context.Context) context.Context {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(src, carrier)

	dst := context.Background()
	dst = otel.GetTextMapPropagator().Extract(dst, carrier)

	return dst
}

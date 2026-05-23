package trace_util

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer = &Tracer{}

type Tracer struct {
	Tracer trace.TracerProvider
}

func GetTracer() *Tracer {
	return tracer
}

func InitTracer() error {
	tp := initDefaultTracerProvider()

	tracer.Tracer = tp
	otel.SetTracerProvider(tp)
	// 设置文本传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return nil
}

func GetTraceID(ctx context.Context) string {
	// 从上下文中获取 SpanContext（如果存在的话）
	spanCtx := trace.SpanContextFromContext(ctx)
	return spanCtx.TraceID().String()
}

func initDefaultTracerProvider() trace.TracerProvider {
	return sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
}

//func initTracerProvider() trace.TracerProvider {
//	exporter, err := otlptrace.New(
//		context.Background(),
//		otlptracehttp.NewClient(
//			otlptracehttp.WithEndpoint(endpoint),
//			otlptracehttp.WithInsecure(),
//		),
//	)
//	if err != nil {
//		return nil, err
//	}
//
//	res, err := resource.Merge(
//		resource.Default(),
//		resource.NewWithAttributes(
//			semconv.SchemaURL,
//			semconv.ServiceNameKey.String(ServiceName),
//		),
//	)
//	if err != nil {
//		log.Printf("[oteltrace] resource.Merge failed: %v, using fallback resource", err)
//		res = resource.NewWithAttributes(
//			semconv.SchemaURL,
//			semconv.ServiceNameKey.String(ServiceName),
//		)
//	}
//
//	return sdktrace.NewTracerProvider(
//		sdktrace.WithBatcher(exporter),
//		sdktrace.WithResource(res),
//	)
//
//}

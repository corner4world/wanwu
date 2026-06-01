package trace_util

import (
	"context"
	"errors"
	"github.com/UnicomAI/wanwu/api/proto/common"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/redis"
	utils "github.com/UnicomAI/wanwu/pkg/util"
	"google.golang.org/protobuf/proto"

	go_redis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	TraceUserPrefix = "trace_user:"
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

// GetTraceUser 获取追踪用户信息
func GetTraceUser(ctx context.Context) (traceInfo *common.TraceInfo, err error) {
	defer utils.PrintPanicStackWithCall(func(panicOccur bool, recoverError error) {
		if panicOccur {
			traceInfo = nil
			err = recoverError
		}
	})
	traceID := GetTraceID(ctx)
	traceKey := TraceUserKey(traceID)
	result := redis.OP().Cli().Get(ctx, traceKey)
	if result.Err() != nil {
		if errors.Is(result.Err(), go_redis.Nil) {
			return nil, nil
		}
		log.Errorf("get trace user failed: %v", result.Err())
		return nil, result.Err()
	}
	data, err := result.Result()
	if err != nil {
		log.Errorf("get trace user result failed: %v", err)
		return nil, err
	}
	traceInfo = &common.TraceInfo{}
	err = proto.Unmarshal([]byte(data), traceInfo)
	if err != nil {
		log.Errorf("unmarshal trace user failed: %v", err)
		return nil, err
	}
	return traceInfo, nil
}

// TraceUserKey 追踪得用户信息
func TraceUserKey(traceID string) string {
	return TraceUserPrefix + traceID
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

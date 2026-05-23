package trace_util

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"net/http"
)

func NewTracerGin(serviceName string) *gin.Engine {
	r := gin.Default()
	// 启用 ContextWithFallback，使 gin.Context.Value() 委托到 Request.Context()
	// 这样才能正确获取 otelgin 中间件设置的 trace context
	r.ContextWithFallback = true
	r.Use(otelgin.Middleware(serviceName))
	return r
}

func NewResty(ctx context.Context) *resty.Client {
	return resty.New().SetPreRequestHook(func(client *resty.Client, request *http.Request) error {
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(request.Header))
		return nil
	})
}

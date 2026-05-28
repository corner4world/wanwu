package interceptor

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/UnicomAI/wanwu/pkg/log"
	trace_util "github.com/UnicomAI/wanwu/pkg/trace-util"
	"google.golang.org/grpc"
)

func LoggingUnaryGRPC() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		startTime := time.Now()
		requestId := trace_util.GetTraceID(ctx)

		// 记录请求
		reqBuf := new(bytes.Buffer)
		if err := json.NewEncoder(reqBuf).Encode(req); err != nil {
			log.Errorf("[Request ID: %s] Request Method %s | Failed to encode request: %v", requestId, info.FullMethod, err)
		}
		log.Infof("[Request ID: %s] Request Method %s | Request Body: %s", requestId, info.FullMethod, reqBuf.String())
		// 将请求ID添加到上下文中，以便下游服务也可以访问它
		//ctx = context.WithValue(ctx, "request_id", requestId)

		// 调用下一个handler
		resp, err := handler(ctx, req)
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		if err != nil {
			log.Errorf("[Request ID: %s] Error handling request: %v", requestId, err)
			return nil, err
		}

		// 记录响应
		respBuf := new(bytes.Buffer)
		if err := json.NewEncoder(respBuf).Encode(resp); err != nil {
			log.Errorf("[Request ID: %s] Failed to encode response: %v", requestId, err)
		}
		log.Infof("[Request ID: %s] Request Method %s | Request Duration: %s, Response Body: %s", requestId, info.FullMethod, duration, respBuf.String())

		return resp, err
	}
}

func LoggingStreamGRPC() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()
		// 从上下文中获取 SpanContext（如果存在的话）
		ctx := ss.Context()
		requestId := trace_util.GetTraceID(ctx)

		log.Infof("[Stream Request ID: %s] Request Method %s | Start", requestId, info.FullMethod)

		// 调用下一个handler
		err := handler(srv, ss)
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		if err != nil {
			log.Errorf("[Stream Request ID: %s] Request Method %s | Error: %v, Duration: %s", requestId, info.FullMethod, err, duration)
			return err
		}

		log.Infof("[Stream Request ID: %s] Request Method %s | Duration: %s", requestId, info.FullMethod, duration)
		return nil
	}
}

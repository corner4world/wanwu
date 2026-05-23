package trace_util

import (
	"fmt"
	"github.com/UnicomAI/wanwu/pkg/log"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"runtime/debug"
)

const (
	headlessServiceSchema = "dns:///"
	maxMsgSize            = 1024 * 1024 * 4 // 4M
)

func NewGrpcTracerServer(interceptors []grpc.UnaryServerInterceptor, streamInterceptors []grpc.StreamServerInterceptor) *grpc.Server {
	// init
	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(func(p interface{}) error {
			log.Errorf("[PANIC] %v\n%v", p, string(debug.Stack()))
			return status.Error(codes.Internal, fmt.Sprintf("panic: %v", p))
		}),
	}
	var interceptorList []grpc.UnaryServerInterceptor
	interceptorList = append(interceptorList, grpc_recovery.UnaryServerInterceptor(opts...))
	if len(interceptors) > 0 {
		interceptorList = append(interceptorList, interceptors...)
	}
	var streamInterceptorList []grpc.StreamServerInterceptor
	streamInterceptorList = append(streamInterceptorList, grpc_recovery.StreamServerInterceptor(opts...))
	if len(streamInterceptors) > 0 {
		streamInterceptorList = append(streamInterceptorList, streamInterceptors...)
	}
	serverOptions := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(interceptorList...),
		grpc.ChainStreamInterceptor(streamInterceptorList...),
	}
	//if len(serverOptionList) > 0 {
	//	serverOptions = append(serverOptions, serverOptionList...)
	//}
	return grpc.NewServer(serverOptions...)
}

func NewGrpcTracerConn(host string, intercepts []grpc.UnaryClientInterceptor) (*grpc.ClientConn, error) {
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMsgSize),
			grpc.MaxCallSendMsgSize(maxMsgSize)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
	if len(intercepts) > 0 {
		options = append(options, grpc.WithChainUnaryInterceptor(intercepts...))
	}
	conn, err := grpc.NewClient(headlessServiceSchema+host,
		options...,
	)
	if err != nil {
		return nil, err
	}
	return conn, err
}

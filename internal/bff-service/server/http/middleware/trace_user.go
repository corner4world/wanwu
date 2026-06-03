package middleware

import (
	"time"

	"github.com/UnicomAI/wanwu/api/proto/common"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/redis"
	trace_util "github.com/UnicomAI/wanwu/pkg/trace-util"
	utils "github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

const (
	TraceAppIDKey                = "trace_app_id"
	GeneralAgentChatApi          = "/v1/general/agent/conversation/chat"
	TraceUserTimeout             = 30 * time.Minute
	TraceUserGeneralAgentTimeout = 6 * time.Hour //6个小时，理论上可以涵盖所有通用智能体会话的超时，如果发现bad case 提高这个值
)

var longTimeoutPathMap = map[string]bool{GeneralAgentChatApi: true}

// TraceUser 追踪的用户信息
func TraceUser(ctx *gin.Context) {
	defer utils.PrintPanicStackWithCall(func(panicOccur bool, recoverError error) {
		if panicOccur {
			log.Infof("trace user panic %v", recoverError)
		}
		ctx.Next()
	})
	//校验是否存储了用户信息，如果存储了则忽略
	user, _ := trace_util.GetTraceUser(ctx)
	if user != nil {
		return
	}
	traceID := trace_util.GetTraceID(ctx)
	data, err := buildTraceData(ctx, traceID)
	if err != nil {
		log.Errorf("build trace data failed: %v", err)
	}
	err = redis.OP().Cli().Set(ctx, trace_util.TraceUserKey(traceID), data, buildKeyTimeout(ctx)).Err()
	if err != nil {
		log.Errorf("set trace user failed: %v", err)
	}
}

func TraceAppID(appIdField string) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		setTraceAppID(ctx, appIdField)
	}
}

// buildKeyTimeout 构造redis key 的过期时间
func buildKeyTimeout(ctx *gin.Context) time.Duration {
	var expiration = TraceUserTimeout
	if longTimeoutPathMap[ctx.Request.URL.Path] {
		expiration = TraceUserGeneralAgentTimeout
	}
	return expiration
}

// buildTraceData 构建追踪数据
func buildTraceData(ctx *gin.Context, traceID string) ([]byte, error) {
	// 1. 创建对象
	info := buildTraceInfo(ctx, traceID)
	// 2. 将对象序列化为字节数组
	data, err := proto.Marshal(info)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// buildTraceInfo 构建追踪信息
func buildTraceInfo(ctx *gin.Context, traceID string) *common.TraceInfo {
	userID, orgID, _ := getUserInfo(ctx)
	var appID = getTraceAppID(ctx)
	return &common.TraceInfo{
		TraceId: traceID,
		TraceUser: &common.TraceUser{
			UserId: userID,
			OrgId:  orgID,
		},
		TraceApp: &common.TraceApp{
			AppId: appID,
		},
		TraceApi: &common.TraceApi{
			ApiPath: ctx.Request.URL.Path,
		},
	}
}

// getUserInfo 获取用户信息
func getUserInfo(ctx *gin.Context) (userID, orgID string, err error) {
	// userID
	userID, err = getUserID(ctx)
	if err != nil {
		return "", "", err
	}

	// orgID
	orgID, err = getOrgID(ctx)
	if err != nil {
		return "", "", err
	}
	return userID, orgID, nil
}

func getTraceAppID(ctx *gin.Context) string {
	return ctx.GetString(TraceAppIDKey)
}

func setTraceAppID(ctx *gin.Context, appIdFileName string) {
	ctx.Set(TraceAppIDKey, getFieldValue(ctx, appIdFileName))
}

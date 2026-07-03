package v1

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// validChannelTypes 允许的通道类型
var validChannelTypes = map[string]bool{
	"wechat":   true,
	"dingtalk": true,
	"feishu":   true,
}

// isValidChannelType 验证通道类型是否合法
func isValidChannelType(channelType string) bool {
	return validChannelTypes[channelType]
}

// --- 万悟平台代理 ---

// ListWanwuAgents 获取万悟智能体列表（含 UUID，供通道绑定使用）
func ListWanwuAgents(ctx *gin.Context) {
	var req request.ListWanwuAgentsRequest
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.ListWanwuAgents(ctx, getUserID(ctx), getOrgID(ctx), req.AppType, req.Name)
	gin_util.Response(ctx, resp, err)
}

// ListWanwuApiKeys 获取万悟 API Key 列表（用于通道选择下拉）
func ListWanwuApiKeys(ctx *gin.Context) {
	resp, err := service.ListWanwuApiKeys(ctx, getUserID(ctx), getOrgID(ctx))
	gin_util.Response(ctx, resp, err)
}

// --- 扫码登录 ---

// CreateQRLogin 发起扫码登录
func CreateQRLogin(ctx *gin.Context) {
	channelType := ctx.Param("channelType")
	if channelType == "" {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_type_required")
		return
	}
	if !isValidChannelType(channelType) {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_type_invalid")
		return
	}
	resp, err := service.CreateQRLogin(ctx, channelType, getUserID(ctx), getOrgID(ctx))
	gin_util.Response(ctx, resp, err)
}

// GetQRLoginStatus 查询扫码状态
func GetQRLoginStatus(ctx *gin.Context) {
	channelType := ctx.Param("channelType")
	sessionID := ctx.Param("sessionId")
	if channelType == "" || sessionID == "" {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_type_and_session_id_required")
		return
	}
	if !isValidChannelType(channelType) {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_type_invalid")
		return
	}
	resp, err := service.GetQRLoginStatus(ctx, channelType, sessionID)
	gin_util.Response(ctx, resp, err)
}

// CancelQRLogin 取消扫码登录
func CancelQRLogin(ctx *gin.Context) {
	channelType := ctx.Param("channelType")
	sessionID := ctx.Param("sessionId")
	if channelType == "" || sessionID == "" {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_type_and_session_id_required")
		return
	}
	if !isValidChannelType(channelType) {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_type_invalid")
		return
	}
	err := service.CancelQRLogin(ctx, channelType, sessionID)
	gin_util.Response(ctx, nil, err)
}

// CompleteQRLogin 完成扫码登录（扫码成功后创建通道）
func CompleteQRLogin(ctx *gin.Context) {
	channelType := ctx.Param("channelType")
	sessionID := ctx.Param("sessionId")
	if channelType == "" || sessionID == "" {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_type_and_session_id_required")
		return
	}
	if !isValidChannelType(channelType) {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_type_invalid")
		return
	}
	resp, err := service.CompleteQRLogin(ctx, channelType, sessionID, getUserID(ctx), getOrgID(ctx))
	gin_util.Response(ctx, resp, err)
}

// --- 通道管理 ---

// CreateChannel 创建通道
func CreateChannel(ctx *gin.Context) {
	var req request.CreateChannelRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	if !isValidChannelType(req.ChannelType) {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_type_invalid")
		return
	}
	resp, err := service.CreateChannel(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// ListChannels 获取通道列表
func ListChannels(ctx *gin.Context) {
	name := ctx.Query("name")
	resp, err := service.ListChannels(ctx, getUserID(ctx), getOrgID(ctx), name, getPageNo(ctx), getPageSize(ctx))
	gin_util.Response(ctx, resp, err)
}

// GetChannel 获取通道详情
func GetChannel(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_id_required")
		return
	}
	resp, err := service.GetChannel(ctx, id)
	gin_util.Response(ctx, resp, err)
}

// UpdateChannel 更新通道
func UpdateChannel(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_id_required")
		return
	}
	var req request.UpdateChannelRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.UpdateChannel(ctx, id, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// UpdateChannelStatus 启用/停用通道
func UpdateChannelStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_id_required")
		return
	}
	var req request.UpdateChannelStatusRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.UpdateChannelStatus(ctx, id, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// DeleteChannel 删除通道
func DeleteChannel(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_id_required")
		return
	}
	err := service.DeleteChannel(ctx, id, getUserID(ctx), getOrgID(ctx))
	gin_util.Response(ctx, nil, err)
}

// DisconnectChannel 断开通道
func DisconnectChannel(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		gin_util.ResponseErrCodeKey(ctx, 110001, "channel_id_required")
		return
	}
	resp, err := service.DisconnectChannel(ctx, id)
	gin_util.Response(ctx, resp, err)
}

package callback

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// AgentChatProxy
//
//	@Tags			callback
//	@Summary		智能体代理问答
//	@Description	智能体代理问答，固定流式返回，提取eventType=0的数据聚合返回
//	@Accept			json
//	@Produce		json
//	@Param			assistantId	path		string						true	"assistantId"
//	@Param			data		body		request.AgentChatProxyReq	true	"智能体代理问答请求参数"
//	@Success		200			{object}	response.Response
//	@Router			/agent/{assistantId}/chat [post]
func AgentChatProxy(ctx *gin.Context) {
	var req request.AgentChatProxyReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	data, err := service.AgentChatProxy(ctx, ctx.Param("assistantId"), &req)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}

	gin_util.Response(ctx, data, err)
}

package callback

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// ChannelSendMessage
//
//	@Tags			channel
//	@Summary		向通道发送消息
//	@Description	向通道发送消息
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.ChannelSendMessageRequest	true	"请求参数"
//	@Success		200		{object}	response.Response{}
//	@Router			/channel/send-message [post]
func ChannelSendMessage(ctx *gin.Context) {
	var req request.ChannelSendMessageRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	if err := service.SendMessage(ctx.Request.Context(), &req); err != nil {
		gin_util.ResponseErr(ctx, err)
		return
	}
	gin_util.ResponseOK(ctx)
}

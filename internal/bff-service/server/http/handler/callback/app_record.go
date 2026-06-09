package callback

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// AppRecord
//
//	@Tags			callback
//	@Summary		应用使用记录
//	@Description	应用使用记录
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.AppRecordRequest	true	"应用使用记录请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/app/record [post]
func AppRecord(ctx *gin.Context) {
	var req request.AppRecordRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	service.RecordAppStatistic(ctx.Request.Context(), req.UserID, req.OrgID, req.AppID, req.AppType, req.IsSuccess, req.IsStream, req.StreamCosts, 0, req.Source)
	gin_util.Response(ctx, nil, nil)
}

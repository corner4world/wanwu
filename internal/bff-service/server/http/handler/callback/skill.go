package callback

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// SearchBuiltInSkillList
//
//	@Tags			skill
//	@Summary		获取内置skill详情列表
//	@Description	获取内置skill详情列表
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.SearchBuiltinSkillListReq	true	"请求参数"
//	@Success		200		{object}	response.Response{data=response.SkillDetailListResp}
//	@Router			/callback/v1/skill/builtin/list [post]
func SearchBuiltInSkillList(ctx *gin.Context) {
	var req request.SearchBuiltinSkillListReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetAgentSkillListDetail(ctx, req.SkillIdList)
	gin_util.Response(ctx, resp, err)
}

// SearchCustomSkillList
//
//	@Tags			skill
//	@Summary		获取自定义skill详情列表
//	@Description	获取自定义skill详情列表
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.SearchCustomSkillListReq	true	"请求参数"
//	@Success		200		{object}	response.Response{data=response.CustomSkillDetailListResp}
//	@Router			/callback/v1/skill/custom/list [post]
func SearchCustomSkillList(ctx *gin.Context) {
	var req request.SearchCustomSkillListReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetCustomSkillListDetail(ctx, req.SkillIdList)
	gin_util.Response(ctx, resp, err)
}

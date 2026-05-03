package v1

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// GetJoinerSkillList 获取我添加的skill列表
//
//	@Tags			resource.skill
//	@Summary		获取我添加的skill列表
//	@Description	获取资源库中我添加的skill列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	false	"skill名称"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.JoinerSkillDetail}}
//	@Router			/agent/joiner/skills [get]
func GetJoinerSkillList(ctx *gin.Context) {
	resp, err := service.GetJoinerSkillList(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("name"))
	gin_util.Response(ctx, resp, err)
}

// DeleteJoinerSkill 删除我添加的skill
//
//	@Tags			resource.skill
//	@Summary		删除我添加的skill
//	@Description	删除资源库中我添加的skill
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			skillId	body		request.DeleteJoinerSkillReq	true	"skill ID"
//	@Success		200		{object}	response.Response
//	@Router			/agent/joiner/skills [delete]
func DeleteJoinerSkill(ctx *gin.Context) {
	var req request.DeleteJoinerSkillReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.DeleteJoinerSkill(ctx, req.SkillId)
	gin_util.Response(ctx, nil, err)
}

// GetJoinerSkillDetail 获取我添加的skill详情
//
//	@Tags			resource.skill
//	@Summary		获取我添加的skill详情
//	@Description	获取资源库中我添加的skill详情
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			skillId	query		string	true	"skill ID"
//	@Success		200		{object}	response.Response{data=response.JoinerSkillDetail}
//	@Router			/agent/joiner/skills/detail [get]
func GetJoinerSkillDetail(ctx *gin.Context) {
	resp, err := service.GetJoinerSkill(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("skillId"))
	gin_util.Response(ctx, resp, err)
}

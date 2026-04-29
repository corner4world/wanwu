package v1

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// GetCustomSkillList
//
//	@Tags			resource.skill
//	@Summary		获取自定义skill列表
//	@Description	获取自定义skill列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	false	"skill名称"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.CustomSkillDetail}}
//	@Router			/agent/skill/custom/list [get]
func GetCustomSkillList(ctx *gin.Context) {
	resp, err := service.GetCustomSkillList(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("name"))
	gin_util.Response(ctx, resp, err)
}

// GetCustomSkillDetail
//
//	@Tags			resource.skill
//	@Summary		获取自定义skill详情
//	@Description	获取自定义skill详情
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			skillId	query		string	true	"skill ID"
//	@Success		200		{object}	response.Response{data=response.CustomSkillDetail}
//	@Router			/agent/skill/custom/detail [get]
func GetCustomSkillDetail(ctx *gin.Context) {
	resp, err := service.GetCustomSkill(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("skillId"))
	gin_util.Response(ctx, resp, err)
}

// CreateCustomSkill
//
//	@Tags			resource.skill
//	@Summary		创建自定义skill
//	@Description	创建自定义skill
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.CreateCustomSkillReq	true	"自定义skill信息"
//	@Success		200		{object}	response.Response{data=response.CustomSkillIDResp}
//	@Router			/agent/skill/custom [post]
func CreateCustomSkill(ctx *gin.Context) {
	var req request.CreateCustomSkillReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.CreateCustomSkill(ctx, getUserID(ctx), getOrgID(ctx), req.Avatar.Key, req.Author, req.ZipUrl, "", "skill_import")
	gin_util.Response(ctx, resp, err)
}

// DeleteCustomSkill
//
//	@Tags			resource.skill
//	@Summary		删除自定义skill
//	@Description	删除自定义skill
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			skillId	body		request.DeleteCustomSkillReq	true	"skill ID"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/custom [delete]
func DeleteCustomSkill(ctx *gin.Context) {
	var req request.DeleteCustomSkillReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.DeleteCustomSkill(ctx, req.SkillId)
	gin_util.Response(ctx, nil, err)
}

// CheckCustomSkill
//
//	@Tags			resource.skill
//	@Summary		校验自定义skill zip包
//	@Description	校验自定义skill zip包是否有效（包含SKILL.md文件）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.CheckCustomSkillReq	true	"zip包URL"
//	@Success		200		{object}	response.Response{data=response.CustomSkillCheckResp}
//	@Router			/agent/skill/custom/check [post]
func CheckCustomSkill(ctx *gin.Context) {
	var req request.CheckCustomSkillReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.CheckCustomSkill(ctx, getUserID(ctx), getOrgID(ctx), req.ZipUrl)
	gin_util.Response(ctx, resp, err)
}

// GetSkillSelect
//
//	@Tags			resource.skill
//	@Summary		获取skill选择列表
//	@Description	获取skill选择列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name		query		string	false	"skill名称"
//	@Param			skillType	query		string	false	"skill类型(builtin/custom)"
//	@Success		200			{object}	response.Response{data=response.ListResult{list=[]response.SkillInfo}}
//	@Router			/agent/skill/select [get]
func GetSkillSelect(ctx *gin.Context) {
	resp, err := service.GetSkillSelect(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("name"), ctx.Query("skillType"))
	gin_util.Response(ctx, resp, err)
}

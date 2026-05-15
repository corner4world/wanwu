package v1

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// GetAcquiredSkillList 获取我添加的skill列表
//
//	@Tags			resource.skill
//	@Summary		获取我添加的skill列表
//	@Description	获取资源库中我添加的skill列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	false	"skill名称"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.AcquiredSkillDetail}}
//	@Router			/agent/acquired/skill/list [get]
func GetAcquiredSkillList(ctx *gin.Context) {
	resp, err := service.GetAcquiredSkillList(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("name"))
	gin_util.Response(ctx, resp, err)
}

// DeleteAcquiredSkill 删除我添加的skill
//
//	@Tags			resource.skill
//	@Summary		删除我添加的skill
//	@Description	删除资源库中我添加的skill
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			skillId	body		request.DeleteAcquiredSkillReq	true	"skill ID"
//	@Success		200		{object}	response.Response
//	@Router			/agent/acquired/skill [delete]
func DeleteAcquiredSkill(ctx *gin.Context) {
	var req request.DeleteAcquiredSkillReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.DeleteAcquiredSkill(ctx, getUserID(ctx), getOrgID(ctx), req.SkillId)
	gin_util.Response(ctx, nil, err)
}

// GetAcquiredSkillDetail 获取我添加的skill详情
//
//	@Tags			resource.skill
//	@Summary		获取我添加的skill详情
//	@Description	获取资源库中我添加的skill详情
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			skillId	query		string	true	"skill ID"
//	@Success		200		{object}	response.Response{data=response.AcquiredSkillDetail}
//	@Router			/agent/acquired/skill/detail [get]
func GetAcquiredSkillDetail(ctx *gin.Context) {
	resp, err := service.GetAcquiredSkill(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("skillId"))
	gin_util.Response(ctx, resp, err)
}

// CreateAcquiredSkillConfig 新增我添加的skill配置
//
//	@Tags			resource.skill
//	@Summary		新增我添加的skill配置
//	@Description	新增资源库中我添加的skill变量配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.SkillConfigReq	true	"skill配置"
//	@Success		200		{object}	response.Response
//	@Router			/agent/acquired/skill/config [post]
func CreateAcquiredSkillConfig(ctx *gin.Context) {
	var req request.SkillConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.CreateAcquiredSkillConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

// UpdateAcquiredSkillConfig 编辑我添加的skill配置
//
//	@Tags			resource.skill
//	@Summary		编辑我添加的skill配置
//	@Description	编辑资源库中我添加的skill变量配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.UpdateSkillConfigReq	true	"skill配置"
//	@Success		200		{object}	response.Response
//	@Router			/agent/acquired/skill/config [put]
func UpdateAcquiredSkillConfig(ctx *gin.Context) {
	var req request.UpdateSkillConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.UpdateAcquiredSkillConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

// DeleteAcquiredSkillConfig 删除我添加的skill配置
//
//	@Tags			resource.skill
//	@Summary		删除我添加的skill配置
//	@Description	删除资源库中我添加的skill变量配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.DeleteSkillConfigReq	true	"skill配置"
//	@Success		200		{object}	response.Response
//	@Router			/agent/acquired/skill/config [delete]
func DeleteAcquiredSkillConfig(ctx *gin.Context) {
	var req request.DeleteSkillConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.DeleteAcquiredSkillConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

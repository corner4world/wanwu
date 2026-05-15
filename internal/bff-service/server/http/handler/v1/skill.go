package v1

import (
	"fmt"
	"net/http"
	"net/url"

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
	err := service.DeleteCustomSkill(ctx, getUserID(ctx), getOrgID(ctx), req.SkillId)
	gin_util.Response(ctx, nil, err)
}

// CreateCustomSkillConfig
//
//	@Tags			resource.skill
//	@Summary		新增自定义skill配置
//	@Description	新增自定义skill变量配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.SkillConfigReq	true	"skill配置"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/custom/config [post]
func CreateCustomSkillConfig(ctx *gin.Context) {
	var req request.SkillConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.CreateCustomSkillConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

// UpdateCustomSkillConfig
//
//	@Tags			resource.skill
//	@Summary		编辑自定义skill配置
//	@Description	编辑自定义skill变量配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.UpdateSkillConfigReq	true	"skill配置"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/custom/config [put]
func UpdateCustomSkillConfig(ctx *gin.Context) {
	var req request.UpdateSkillConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.UpdateCustomSkillConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

// DeleteCustomSkillConfig
//
//	@Tags			resource.skill
//	@Summary		删除自定义skill配置
//	@Description	删除自定义skill变量配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.DeleteSkillConfigReq	true	"skill配置"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/custom/config [delete]
func DeleteCustomSkillConfig(ctx *gin.Context) {
	var req request.DeleteSkillConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.DeleteCustomSkillConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
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

// GetBuiltinSkillList
//
//	@Tags			resource.skill
//	@Summary		获取内置skill列表
//	@Description	获取资源库内置skill列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	false	"skill名称"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.SkillDetail}}
//	@Router			/agent/skill/builtin/list [get]
func GetBuiltinSkillList(ctx *gin.Context) {
	resp, err := service.GetBuiltinSkillList(ctx, ctx.Query("name"))
	gin_util.Response(ctx, resp, err)
}

// GetBuiltinSkillDetail
//
//	@Tags			resource.skill
//	@Summary		获取内置skill详情
//	@Description	获取资源库内置skill详情
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			skillId	query		string	true	"skill ID"
//	@Success		200		{object}	response.Response{data=response.SkillDetail}
//	@Router			/agent/skill/builtin/detail [get]
func GetBuiltinSkillDetail(ctx *gin.Context) {
	resp, err := service.GetBuiltinSkillDetail(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("skillId"))
	gin_util.Response(ctx, resp, err)
}

// DownloadBuiltinSkill
//
//	@Tags			resource.skill
//	@Summary		下载内置skill
//	@Description	下载内置skill ZIP包
//	@Security		JWT
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			skillId	query		string	true	"skill ID"
//	@Success		200		{object}	response.Response
//	@Router			/builtin/skill/download [get]
func DownloadBuiltinSkill(ctx *gin.Context) {
	skillId := ctx.Query("skillId")
	fileName := fmt.Sprintf("%s.zip", skillId)
	resp, err := service.DownloadBuiltinSkill(ctx, skillId)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Data(http.StatusOK, "application/octet-stream", resp)
}

// CreateBuiltinSkillConfig
//
//	@Tags			resource.skill
//	@Summary		新增内置skill配置
//	@Description	新增内置skill变量配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.SkillConfigReq	true	"skill配置"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/builtin/config [post]
func CreateBuiltinSkillConfig(ctx *gin.Context) {
	var req request.SkillConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.CreateBuiltinSkillConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

// UpdateBuiltinSkillConfig
//
//	@Tags			resource.skill
//	@Summary		编辑内置skill配置
//	@Description	编辑内置skill变量配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.UpdateSkillConfigReq	true	"skill配置"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/builtin/config [put]
func UpdateBuiltinSkillConfig(ctx *gin.Context) {
	var req request.UpdateSkillConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.UpdateBuiltinSkillConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

// DeleteBuiltinSkillConfig
//
//	@Tags			resource.skill
//	@Summary		删除内置skill配置
//	@Description	删除内置skill变量配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.DeleteSkillConfigReq	true	"skill配置"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/builtin/config [delete]
func DeleteBuiltinSkillConfig(ctx *gin.Context) {
	var req request.DeleteSkillConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.DeleteBuiltinSkillConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

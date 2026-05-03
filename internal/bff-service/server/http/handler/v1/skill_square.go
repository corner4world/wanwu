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

// GetSquareSkillList 获取广场skill列表
//
//	@Tags			exploration.skill
//	@Summary		获取广场skill列表
//	@Description	获取探索广场中的skill列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	false	"skill名称"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.SquareSkillDetail}}
//	@Router			/square/skills [get]
func GetSquareSkillList(ctx *gin.Context) {
	resp, err := service.GetSquareSkillList(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("name"))
	gin_util.Response(ctx, resp, err)
}

// ShareSquareSkill 添加广场skill到资源库
//
//	@Tags			exploration.skill
//	@Summary		添加广场skill到资源库
//	@Description	将广场中的skill添加到我的资源库
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			skillId	body		request.ShareSquareSkillReq	true	"skill ID"
//	@Success		200		{object}	response.Response
//	@Router			/square/skills/share [post]
func ShareSquareSkill(ctx *gin.Context) {
	var req request.ShareSquareSkillReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.ShareSquareSkill(ctx, getUserID(ctx), getOrgID(ctx), req.SkillId)
	gin_util.Response(ctx, nil, err)
}

// GetSquareSkillDetail 获取广场skill详情
//
//	@Tags			exploration.skill
//	@Summary		获取广场skill详情
//	@Description	获取探索广场中的skill详情
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			skillId	query		string	true	"skill ID"
//	@Success		200		{object}	response.Response{data=response.SquareSkillDetailInfo}
//	@Router			/square/skills/detail [get]
func GetSquareSkillDetail(ctx *gin.Context) {
	resp, err := service.GetSquareSkillDetail(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("skillId"))
	gin_util.Response(ctx, resp, err)
}

// DownloadSquareSkill 下载广场skill
//
//	@Tags			exploration.skill
//	@Summary		下载广场skill
//	@Description	下载探索广场中的skill ZIP包
//	@Security		JWT
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			skillId	query		string	true	"skill ID"
//	@Success		200		{object}	response.Response
//	@Router			/square/skills/download [get]
func DownloadSquareSkill(ctx *gin.Context) {
	fileName := fmt.Sprintf("%s.zip", ctx.Query("skillId"))
	resp, err := service.DownloadSquareSkill(ctx, ctx.Query("skillId"))
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Data(http.StatusOK, "application/octet-stream", resp)
}

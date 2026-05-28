package v1

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// GetOrgsStatisticSelect
//
//	@Tags			app_observability.statistic
//	@Summary		获取统计看板组织下拉列表
//	@Description	系统组织管理员返回全部组织及下级；组织管理员返回当前组织及下级；普通用户仅当前组织
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=response.ListResult{list=[]response.IDName}}
//	@Router			/statistic/orgs/select [get]
func GetOrgsStatisticSelect(ctx *gin.Context) {
	resp, err := service.GetOrgsStatisticSelect(ctx, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isAdmin(ctx) && isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}

// GetUsersStatisticSelect
//
//	@Tags			app_observability.statistic
//	@Summary		获取统计看板用户下拉列表
//	@Description	组织/系统管理员：以 JWT orgId 为根，返回该组织及全部下级组织下的用户（忽略 body.orgIds/userIds）；普通用户仅返回本人
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=response.ListResult{list=[]response.BriefUserInfo}}
//	@Router			/statistic/users/select [get]
func GetUsersStatisticSelect(ctx *gin.Context) {
	resp, err := service.GetUsersStatisticSelect(ctx, getUserID(ctx), getOrgID(ctx), isAdmin(ctx))
	gin_util.Response(ctx, resp, err)
}

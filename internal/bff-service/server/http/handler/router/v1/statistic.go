package v1

import (
	"net/http"

	v1 "github.com/UnicomAI/wanwu/internal/bff-service/server/http/handler/v1"
	mid "github.com/UnicomAI/wanwu/pkg/gin-util/mid-wrap"
	"github.com/gin-gonic/gin"
)

func registerStatistic(apiV1 *gin.RouterGroup) {
	// filter & view
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/orgs/select", http.MethodGet, v1.GetOrgsStatisticSelect, "获取统计看板组织下拉列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/users/select", http.MethodGet, v1.GetUsersStatisticSelect, "获取统计看板用户下拉列表")

	// app
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/app", http.MethodPost, v1.GetAppStatistic, "获取应用统计数据")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/app/list", http.MethodPost, v1.GetAppStatisticList, "获取应用统计列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/app/export", http.MethodPost, v1.ExportAppStatisticList, "导出应用统计列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/app/select", http.MethodPost, v1.GetAppListSelect, "获取应用下拉列表")

	// model
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/model", http.MethodPost, v1.GetModelStatistic, "获取模型统计数据")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/model/list", http.MethodPost, v1.GetModelStatisticList, "获取模型统计列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/model/export", http.MethodPost, v1.ExportModelStatisticList, "导出模型统计列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/model/select", http.MethodPost, v1.GetStatisticModelSelect, "获取模型统计下拉列表")

	// api key
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api", http.MethodPost, v1.GetAPIKeyStatistic, "获取API Key统计数据")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/list", http.MethodPost, v1.GetAPIKeyStatisticList, "获取API Key调用统计")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/record", http.MethodPost, v1.GetAPIKeyStatisticRecord, "获取API Key调用记录")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/list/export", http.MethodPost, v1.ExportAPIKeyStatisticList, "导出API Key统计列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/record/export", http.MethodPost, v1.ExportAPIKeyStatisticRecord, "导出API Key调用记录")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/routes", http.MethodGet, v1.GetApiKeyStatisticRoutes, "获取API Key统计路由列表")
	mid.Sub("app_observability.statistic").Reg(apiV1, "/statistic/api/select", http.MethodPost, v1.GetStatisticAPIKeySelect, "获取API Key下拉列表")
}

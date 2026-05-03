package openapi

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/gin-gonic/gin"
)

// ListModels
//
//	@Tags			openapi
//	@Summary		模型列表查询OpenAPI
//	@Description	根据查询条件返回当前 API Key 所属用户/组织可见的模型列表。未传 isActive 时默认仅返回已启用模型。
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			modelType	query		string	false	"模型类型"	Enums(llm,embedding,rerank)
//	@Param			provider	query		string	false	"模型供应商"
//	@Param			displayName	query		string	false	"模型显示名称"
//	@Success		200			{object}	response.Response{data=response.ListResult{list=[]response.OpenAPIModelListItem}}
//	@Router			/model/list [get]
func ListModels(ctx *gin.Context) {
	var req request.ListModelsRequest
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	full, err := service.ListModels(ctx, getUserID(ctx), getOrgID(ctx), &req)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	out := toOpenAPIModelListResult(full)
	gin_util.Response(ctx, out, nil)
}

func toOpenAPIModelListResult(full *response.ListResult) *response.ListResult {
	if full == nil {
		return &response.ListResult{List: []response.OpenAPIModelListItem{}, Total: 0}
	}
	models, ok := full.List.([]*response.ModelInfo)
	if !ok {
		log.Warnf("[OpenAPI][ModelList] unexpected list type: %T, total=%d", full.List, full.Total)
		return &response.ListResult{List: []response.OpenAPIModelListItem{}, Total: full.Total}
	}
	items := make([]response.OpenAPIModelListItem, 0, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		items = append(items, response.OpenAPIModelListItem{
			UUID:        m.Uuid,
			DisplayName: m.DisplayName,
			Provider:    m.Provider,
			ModelType:   m.ModelType,
			Model:       m.Model,
			ScopeType:   m.ScopeType,
		})
	}
	return &response.ListResult{List: items, Total: full.Total}
}

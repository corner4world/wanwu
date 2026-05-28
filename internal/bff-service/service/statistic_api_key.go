package service

import (
	app_service "github.com/UnicomAI/wanwu/api/proto/app-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type apiKeyInfo struct {
	name string
	key  string
}

func GetAPIKeyStatistic(ctx *gin.Context, req *request.APIKeyStatisticReq, userId, orgId string, isAdmin, isSystem bool) (*response.APIKeyStatistic, error) {
	scope, err := ResolveStatisticScope(ctx, req.StatisticFilter, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}
	resp, err := app.GetAPIKeyStatistic(ctx.Request.Context(), &app_service.GetAPIKeyStatisticReq{
		OrgIds:      scope.OrgIds,
		UserIds:     scope.UserIds,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		ApiKeyIds:   normalizeAPIKeyIds(req.APIKeyIds),
		MethodPaths: req.MethodPaths,
	})
	if err != nil {
		return nil, err
	}
	return &response.APIKeyStatistic{
		Overview: response.APIKeyStatisticOverview{
			CallCount:         convertAPIKeyStatisticOverviewItem(resp.Overview.CallCount),
			CallFailure:       convertAPIKeyStatisticOverviewItem(resp.Overview.CallFailure),
			AvgStreamCosts:    convertAPIKeyStatisticOverviewItem(resp.Overview.AvgStreamCosts),
			AvgNonStreamCosts: convertAPIKeyStatisticOverviewItem(resp.Overview.AvgNonStreamCosts),
			StreamCount:       convertAPIKeyStatisticOverviewItem(resp.Overview.StreamCount),
			NonStreamCount:    convertAPIKeyStatisticOverviewItem(resp.Overview.NonStreamCount),
		},
		Trend: response.APIKeyStatisticTrend{
			APICalls: convertStatisticChart(ctx, resp.Trend.ApiCalls),
		},
	}, nil
}

func GetAPIKeyStatisticList(ctx *gin.Context, req *request.APIKeyStatisticListReq, userId, orgId string, isAdmin, isSystem bool) (*response.PageResult, error) {
	scope, err := ResolveStatisticScope(ctx, req.StatisticFilter, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}
	resp, err := app.GetAPIKeyStatisticList(ctx.Request.Context(), &app_service.GetAPIKeyStatisticListReq{
		OrgIds:      scope.OrgIds,
		UserIds:     scope.UserIds,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		ApiKeyIds:   normalizeAPIKeyIds(req.APIKeyIds),
		MethodPaths: req.MethodPaths,
		PageNo:      int32(req.PageNo),
		PageSize:    int32(req.PageSize),
	})
	if err != nil {
		return nil, err
	}
	items, err := buildAPIKeyStatisticItems(ctx, scope, resp.Items)
	if err != nil {
		return nil, err
	}
	return &response.PageResult{
		List:     items,
		Total:    int64(resp.Total),
		PageNo:   req.PageNo,
		PageSize: req.PageSize,
	}, nil
}

func GetAPIKeyStatisticRecord(ctx *gin.Context, req *request.APIKeyStatisticRecordReq, userId, orgId string, isAdmin, isSystem bool) (*response.PageResult, error) {
	scope, err := ResolveStatisticScope(ctx, req.StatisticFilter, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}
	resp, err := app.GetAPIKeyStatisticRecord(ctx.Request.Context(), &app_service.GetAPIKeyStatisticRecordReq{
		OrgIds:      scope.OrgIds,
		UserIds:     scope.UserIds,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		ApiKeyIds:   normalizeAPIKeyIds(req.APIKeyIds),
		MethodPaths: req.MethodPaths,
		PageNo:      int32(req.PageNo),
		PageSize:    int32(req.PageSize),
	})
	if err != nil {
		return nil, err
	}
	items, err := buildAPIKeyStatisticRecordItems(ctx, scope, resp.Items)
	if err != nil {
		return nil, err
	}
	return &response.PageResult{
		List:     items,
		Total:    int64(resp.Total),
		PageNo:   req.PageNo,
		PageSize: req.PageSize,
	}, nil
}

func ExportAPIKeyStatisticList(ctx *gin.Context, req *request.APIKeyStatisticReq, userId, orgId string, isAdmin, isSystem bool) (*excelize.File, error) {
	scope, err := ResolveStatisticScope(ctx, req.StatisticFilter, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}

	resp, err := app.GetAPIKeyStatisticList(ctx.Request.Context(), &app_service.GetAPIKeyStatisticListReq{
		OrgIds:      scope.OrgIds,
		UserIds:     scope.UserIds,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		ApiKeyIds:   normalizeAPIKeyIds(req.APIKeyIds),
		MethodPaths: req.MethodPaths,
		PageNo:      -1,
		PageSize:    -1,
	})
	if err != nil {
		return nil, err
	}
	items, err := buildAPIKeyStatisticItems(ctx, scope, resp.Items)
	if err != nil {
		return nil, err
	}
	return writeAPIKeyStatisticListExcel(items)
}

func ExportAPIKeyStatisticRecord(ctx *gin.Context, req *request.APIKeyStatisticReq, userId, orgId string, isAdmin, isSystem bool) (*excelize.File, error) {
	scope, err := ResolveStatisticScope(ctx, req.StatisticFilter, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}
	resp, err := app.GetAPIKeyStatisticRecord(ctx.Request.Context(), &app_service.GetAPIKeyStatisticRecordReq{
		OrgIds:      scope.OrgIds,
		UserIds:     scope.UserIds,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		ApiKeyIds:   normalizeAPIKeyIds(req.APIKeyIds),
		MethodPaths: req.MethodPaths,
		PageNo:      -1,
		PageSize:    -1,
	})
	if err != nil {
		return nil, err
	}
	items, err := buildAPIKeyStatisticRecordItems(ctx, scope, resp.Items)
	if err != nil {
		return nil, err
	}
	return writeAPIKeyStatisticRecordExcel(items)
}

func GetStatisticAPIKeySelect(ctx *gin.Context, filter request.StatisticFilter, userId, orgId string, isAdmin, isSystem bool) (*response.ListResult, error) {
	scope, err := ResolveStatisticScope(ctx, filter, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}
	resp, err := app.ListApiKeys(ctx.Request.Context(), &app_service.ListApiKeysReq{
		OrgIds:   scope.OrgIds,
		UserIds:  scope.UserIds,
		PageNo:   1,
		PageSize: 1000,
	})
	if err != nil {
		return nil, err
	}
	items := make([]response.APIKeyDetailResponse, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, response.APIKeyDetailResponse{
			KeyID: item.KeyId,
			Name:  item.Name,
			Key:   item.Key,
		})
	}
	return &response.ListResult{
		List: items,
	}, nil
}

func RecordAPIKeyCall(ctx *gin.Context, userId, orgId, apiKeyId, methodPath string,
	callTime int64, httpStatus string, isStream bool, streamCosts, nonStreamCosts int64, requestBody, responseBody string) {
	_, err := app.RecordAPIKeyStatistic(ctx.Request.Context(), &app_service.RecordAPIKeyStatisticReq{
		UserId:         userId,
		OrgId:          orgId,
		ApiKeyId:       apiKeyId,
		MethodPath:     methodPath,
		CallTime:       callTime,
		HttpStatus:     httpStatus,
		IsStream:       isStream,
		StreamCosts:    streamCosts,
		NonStreamCosts: nonStreamCosts,
		RequestBody:    requestBody,
		ResponseBody:   responseBody,
	})
	if err != nil {
		log.Errorf("record api key[%v] method_path[%v] call err: %v", apiKeyId, methodPath, err)
	}
}

// --- internal ---
func convertAPIKeyStatisticOverviewItem(item *app_service.APIKeyStatisticOverviewItem) response.StatisticOverviewItem {
	if item == nil {
		return response.StatisticOverviewItem{}
	}
	return response.StatisticOverviewItem{
		Value:            item.Value,
		PeriodOverPeriod: item.PeriodOverPeriod,
	}
}

func buildAPIKeyStatisticItems(ctx *gin.Context, scope *statisticScope, protoItems []*app_service.APIKeyStatisticItem) ([]response.APIKeyStatisticItem, error) {
	infoMap := getAPIKeyInfoMap(ctx, scope)
	var orgIDs []string
	var userIDs []string
	for _, item := range protoItems {
		orgIDs = append(orgIDs, item.OrgId)
		userIDs = append(userIDs, item.UserId)
	}
	orgNameMap, userNameMap, err := buildStatisticOrgUserNameMaps(ctx, orgIDs, userIDs)
	if err != nil {
		return nil, err
	}
	items := make([]response.APIKeyStatisticItem, 0, len(protoItems))
	for _, item := range protoItems {
		info := getAPIKeyDisplayInfo(infoMap, item.ApiKeyId)
		items = append(items, response.APIKeyStatisticItem{
			Name:              info.name,
			APIKey:            info.key,
			MethodPath:        item.MethodPath,
			OrgName:           orgNameMap[item.OrgId],
			UserName:          userNameMap[item.UserId],
			CallCount:         item.CallCount,
			CallFailure:       item.CallFailure,
			AvgStreamCosts:    item.AvgStreamCosts,
			AvgNonStreamCosts: item.AvgNonStreamCosts,
			StreamCount:       item.StreamCount,
			NonStreamCount:    item.NonStreamCount,
		})
	}
	return items, nil
}

func buildAPIKeyStatisticRecordItems(ctx *gin.Context, scope *statisticScope, protoItems []*app_service.APIKeyStatisticRecordItem) ([]response.APIKeyStatisticRecordItem, error) {
	infoMap := getAPIKeyInfoMap(ctx, scope)
	var orgIds []string
	var userIDs []string
	for _, item := range protoItems {
		orgIds = append(orgIds, item.OrgId)
		userIDs = append(userIDs, item.UserId)
	}
	orgNameMap, userNameMap, err := buildStatisticOrgUserNameMaps(ctx, orgIds, userIDs)
	if err != nil {
		return nil, err
	}
	items := make([]response.APIKeyStatisticRecordItem, 0, len(protoItems))
	for _, item := range protoItems {
		info := getAPIKeyDisplayInfo(infoMap, item.ApiKeyId)
		items = append(items, response.APIKeyStatisticRecordItem{
			Name:           info.name,
			APIKey:         info.key,
			MethodPath:     item.MethodPath,
			OrgName:        orgNameMap[item.OrgId],
			UserName:       userNameMap[item.UserId],
			CallTime:       util.Time2Str(item.CallTime),
			ResponseStatus: item.ResponseStatus,
			StreamCosts:    item.StreamCosts,
			NonStreamCosts: item.NonStreamCosts,
			RequestBody:    item.RequestBody,
			ResponseBody:   item.ResponseBody,
		})
	}
	return items, nil
}

func writeAPIKeyStatisticListExcel(items []response.APIKeyStatisticItem) (*excelize.File, error) {
	sheet := "API Key统计列表"
	title := []any{"API Key名称", "API Key", "组织", "用户", "请求路径", "调用次数(次)", "调用失败次数(次)", "流式平均耗时(ms)", "非流式平均耗时(ms)", "调用次数(流式)(次)", "调用次数(非流式)(次)"}
	var rows [][]any
	for _, item := range items {
		rows = append(rows, []any{
			item.Name,
			item.APIKey,
			item.OrgName,
			item.UserName,
			item.MethodPath,
			item.CallCount,
			item.CallFailure,
			item.AvgStreamCosts,
			item.AvgNonStreamCosts,
			item.StreamCount,
			item.NonStreamCount,
		})
	}
	return writeExcel(sheet, title, rows)
}

func writeAPIKeyStatisticRecordExcel(items []response.APIKeyStatisticRecordItem) (*excelize.File, error) {
	sheet := "API Key调用记录"
	title := []any{"API Key名称", "API Key", "组织", "用户", "请求路径", "调用时间", "响应状态", "流式耗时(ms)", "非流式耗时(ms)", "请求体", "响应体"}
	var rows [][]any
	for _, item := range items {
		rows = append(rows, []any{
			item.Name,
			item.APIKey,
			item.OrgName,
			item.UserName,
			item.MethodPath,
			item.CallTime,
			item.ResponseStatus,
			item.StreamCosts,
			item.NonStreamCosts,
			item.RequestBody,
			item.ResponseBody,
		})
	}
	return writeExcel(sheet, title, rows)
}

// normalizeAPIKeyIds apiKeyIds 为 ["ALL"] 时不按 key 过滤（与 StatisticFilter 的 ALL 无关）。
func normalizeAPIKeyIds(ids []string) []string {
	if len(ids) == 1 && ids[0] == "ALL" {
		return nil
	}
	return ids
}

func getAPIKeyInfoMap(ctx *gin.Context, scope *statisticScope) map[string]apiKeyInfo {
	resp, err := app.ListApiKeys(ctx.Request.Context(), &app_service.ListApiKeysReq{
		OrgIds:   scope.OrgIds,
		UserIds:  scope.UserIds,
		PageNo:   -1,
		PageSize: -1,
	})
	if err != nil {
		log.Warnf("get api key info map err: %v", err)
		return nil
	}
	infoMap := make(map[string]apiKeyInfo)
	for _, item := range resp.Items {
		infoMap[item.KeyId] = apiKeyInfo{
			name: item.Name,
			key:  item.Key,
		}
	}
	return infoMap
}

func getAPIKeyDisplayInfo(infoMap map[string]apiKeyInfo, apiKeyID string) apiKeyInfo {
	if info, ok := infoMap[apiKeyID]; ok {
		return info
	}
	return apiKeyInfo{
		name: "该API Key已被删除",
		key:  "该API Key已被删除",
	}
}

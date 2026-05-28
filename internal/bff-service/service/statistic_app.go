package service

import (
	"context"
	"fmt"
	"math"

	app_service "github.com/UnicomAI/wanwu/api/proto/app-service"
	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	"github.com/UnicomAI/wanwu/api/proto/common"
	err_code "github.com/UnicomAI/wanwu/api/proto/err-code"
	rag_service "github.com/UnicomAI/wanwu/api/proto/rag-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/pkg/constant"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func GetAppStatistic(ctx *gin.Context, filter request.StatisticFilter, startDate, endDate string, appIds []string, appType string, userId, orgId string, isAdmin, isSystem bool) (*response.AppStatistic, error) {
	scope, err := ResolveStatisticScope(ctx, filter, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}
	if appType == "" {
		appType = constant.AppTypeAgent
	}
	resp, err := app.GetAppStatistic(ctx.Request.Context(), &app_service.GetAppStatisticReq{
		OrgIds:    scope.OrgIds,
		UserIds:   scope.UserIds,
		StartDate: startDate,
		EndDate:   endDate,
		AppIds:    appIds,
		AppType:   appType,
	})
	if err != nil {
		return nil, err
	}
	return &response.AppStatistic{
		Overview: response.AppStatisticOverview{
			CallCount:         convertModelStatisticOverviewItem(resp.Overview.GetCallCount()),
			CallFailure:       convertModelStatisticOverviewItem(resp.Overview.GetCallFailure()),
			StreamCount:       convertModelStatisticOverviewItem(resp.Overview.GetStreamCount()),
			NonStreamCount:    convertModelStatisticOverviewItem(resp.Overview.GetNonStreamCount()),
			AvgStreamCosts:    convertModelStatisticOverviewItem(resp.Overview.GetAvgStreamCosts()),
			AvgNonStreamCosts: convertModelStatisticOverviewItem(resp.Overview.GetAvgNonStreamCosts()),
		},
		Trend: response.AppStatisticTrend{
			CallTrend: convertStatisticChart(ctx, resp.Trend.GetCallTrend()),
		},
	}, nil
}

func GetAppStatisticList(ctx *gin.Context, filter request.StatisticFilter, startDate, endDate string, appIds []string, appType string, page, pageSize int32, userId, orgId string, isAdmin, isSystem bool) (*response.PageResult, error) {
	scope, err := ResolveStatisticScope(ctx, filter, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}
	if appType == "" {
		appType = constant.AppTypeAgent
	}
	resp, err := app.GetAppStatisticList(ctx.Request.Context(), &app_service.GetAppStatisticListReq{
		OrgIds:    scope.OrgIds,
		UserIds:   scope.UserIds,
		StartDate: startDate,
		EndDate:   endDate,
		AppIds:    appIds,
		AppType:   appType,
		PageNo:    page,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}

	var orgIDs []string
	var statAppIds []string
	var userIDs []string
	for _, item := range resp.Items {
		orgIDs = append(orgIDs, item.OrgId)
		statAppIds = append(statAppIds, item.AppId)
		userIDs = append(userIDs, item.UserId)
	}
	orgNameMap, userNameMap, err := buildStatisticOrgUserNameMaps(ctx, orgIDs, userIDs)
	if err != nil {
		return nil, err
	}
	// 先根据ids获取应用名称，避免在循环中调用接口
	ret, err := getAppNameMap(ctx, statAppIds, appType)
	if err != nil {
		log.Errorf("get app name map err: %v", err)
		return nil, err
	}
	items := make([]response.AppStatisticItem, 0, len(resp.Items))
	for _, item := range resp.Items {
		roundedFailureRate := float32(math.Round(float64(item.FailureRate)*100) / 100)
		roundedAvgStreamCosts := float32(math.Round(float64(item.AvgStreamCosts)*100) / 100)
		roundedAvgNonStreamCosts := float32(math.Round(float64(item.AvgNonStreamCosts)*100) / 100)
		items = append(items, response.AppStatisticItem{
			AppId:             item.AppId,
			AppType:           item.AppType,
			AppName:           getAppDisplayName(ret, item.AppId),
			OrgName:           orgNameMap[item.OrgId],
			UserName:          userNameMap[item.UserId],
			CallCount:         item.CallCount,
			CallFailure:       item.CallFailure,
			FailureRate:       roundedFailureRate,
			StreamCount:       item.StreamCount,
			NonStreamCount:    item.NonStreamCount,
			AvgStreamCosts:    roundedAvgStreamCosts,
			AvgNonStreamCosts: roundedAvgNonStreamCosts,
		})
	}
	return &response.PageResult{
		List:     items,
		Total:    int64(resp.Total),
		PageNo:   int(page),
		PageSize: int(pageSize),
	}, nil
}

func ExportAppStatisticList(ctx *gin.Context, filter request.StatisticFilter, startDate, endDate string, appIds []string, appType string, userId, orgId string, isAdmin, isSystem bool) (*excelize.File, error) {
	resp, err := GetAppStatisticList(ctx, filter, startDate, endDate, appIds, appType, -1, -1, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}
	return writeAppListExcel(resp.List.([]response.AppStatisticItem))
}

func writeAppListExcel(items []response.AppStatisticItem) (*excelize.File, error) {
	sheet := "应用统计列表"
	title := []any{"应用名称", "应用类型", "组织", "用户", "调用次数(次)", "调用失败次数(次)", "失败率(%)", "流式调用次数(次)", "非流式调用次数(次)", "平均首响应耗时(流式)(ms)", "平均耗时(非流式)(ms)"}
	var rows [][]any
	for _, item := range items {
		rows = append(rows, []any{
			item.AppName,
			item.AppType,
			item.OrgName,
			item.UserName,
			item.CallCount,
			item.CallFailure,
			item.FailureRate,
			item.StreamCount,
			item.NonStreamCount,
			item.AvgStreamCosts,
			item.AvgNonStreamCosts,
		})
	}
	return writeExcel(sheet, title, rows)
}

func getAppNameMap(ctx *gin.Context, appId []string, appType string) (map[string]string, error) {
	switch appType {
	case constant.AppTypeAgent:
		agentListInfos, err := assistant.GetAssistantByIds(ctx.Request.Context(), &assistant_service.GetAssistantByIdsReq{AssistantIdList: appId})
		if err != nil {
			log.Errorf("get agent info err: %v", err)
			return nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, fmt.Sprintf("get app name error: %v", err))
		}
		if len(agentListInfos.AssistantInfos) == 0 {
			return make(map[string]string), nil
		}
		// 返回和appId映射，以id为key，name为值的map
		result := make(map[string]string)
		for _, info := range agentListInfos.AssistantInfos {
			result[info.Info.AppId] = info.Info.Name
		}
		return result, nil
	case constant.AppTypeRag:
		ragListInfos, err := rag.GetRagByIds(ctx.Request.Context(), &rag_service.GetRagByIdsReq{RagIdList: appId})
		if err != nil {
			log.Errorf("get rag info err: %v", err)
			return nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, fmt.Sprintf("get app name error: %v", err))
		}
		if len(ragListInfos.RagInfos) == 0 {
			return make(map[string]string), nil
		}
		// 返回和appId映射，以id为key，name为值的map
		result := make(map[string]string)
		for _, info := range ragListInfos.RagInfos {
			result[info.AppId] = info.Name
		}
		return result, nil
	case constant.AppTypeWorkflow, constant.AppTypeChatflow:
		workflowRet, err := ListWorkflowByIDs(ctx, "", appId)
		if err != nil {
			log.Errorf("get workflow info err: %v", err)
			return nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, fmt.Sprintf("get app name error: %v", err))
		}
		// 返回和appId映射，以id为key，name为值的map
		result := make(map[string]string)
		for _, info := range workflowRet.Workflows {
			result[info.WorkflowId] = info.Name
		}
		return result, nil
	}
	return nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, fmt.Sprintf("unsupported app type: %v", appType))
}

func RecordAppStatistic(ctx context.Context, userId, orgId, appId, appType string, isSuccess, isStream bool, streamCosts, nonStreamCosts int64, source string) {
	go func() {
		// 不使用外部ctx，避免外部ctx过期导致统计记录失败
		defer util.PrintPanicStack()
		// 修复bug,需要去app-service查询创建者的userId和orgId
		appInfo, err := app.GetAppInfo(context.Background(), &app_service.GetAppInfoReq{
			AppId:   appId,
			AppType: appType,
		})
		if err != nil {
			log.Errorf("get app info err: %v", err)
			return
		}
		_, err = app.RecordAppStatistic(context.Background(), &app_service.RecordAppStatisticReq{
			UserId:         appInfo.UserId,
			OrgId:          appInfo.OrgId,
			AppId:          appId,
			AppType:        appType,
			IsSuccess:      isSuccess,
			IsStream:       isStream,
			StreamCosts:    streamCosts,
			NonStreamCosts: nonStreamCosts,
			Source:         source,
		})
		if err != nil {
			log.Errorf("record app %v type %v source %v statistic err: %v", appId, appType, source, err)
		}
	}()
}

func GetAppListSelect(ctx *gin.Context, filter request.StatisticFilter, appType string, userId, orgId string, isAdmin, isSystem bool) (*response.ListResult, error) {
	scope, err := ResolveStatisticScope(ctx, filter, userId, orgId, isAdmin, isSystem)
	if err != nil {
		return nil, err
	}
	// 和前端约定好如果不传appType参数，默认查询agent类型的应用列表
	if appType == "" {
		appType = constant.AppTypeAgent
	}
	resp, err := app.GetAppList(ctx.Request.Context(), &app_service.GetAppListReq{
		OrgIds:  scope.OrgIds,
		UserIds: scope.UserIds,
		AppType: appType,
	})
	if err != nil {
		return nil, err
	}

	var appIds []string
	for _, info := range resp.Infos {
		appIds = append(appIds, info.AppId)
	}

	items := make([]response.MyAppItem, 0, len(resp.Infos))

	switch appType {
	case constant.AppTypeAgent:
		agentInfos, err := assistant.GetAssistantByIds(ctx.Request.Context(), &assistant_service.GetAssistantByIdsReq{
			AssistantIdList: appIds,
		})
		if err != nil {
			log.Errorf("app select get agent info err: %v", err)
			return nil, err
		}
		agentMap := make(map[string]*common.AppBrief)
		for _, info := range agentInfos.AssistantInfos {
			if info.Info != nil {
				agentMap[info.Info.AppId] = info.Info
			}
		}
		for _, info := range resp.Infos {
			item := response.MyAppItem{
				AppId:       info.AppId,
				AppType:     info.AppType,
				PublishType: info.PublishType,
				CreatedAt:   info.CreatedAt,
			}
			if agentInfo, ok := agentMap[info.AppId]; ok {
				item.Name = agentInfo.Name
				item.Avatar = cacheAppAvatar(ctx, agentInfo.AvatarPath, appType)
			}
			items = append(items, item)
		}

	case constant.AppTypeRag:
		ragInfos, err := rag.GetRagByIds(ctx.Request.Context(), &rag_service.GetRagByIdsReq{
			RagIdList: appIds,
		})
		if err != nil {
			log.Errorf("app select get rag info err: %v", err)
			return nil, err
		}
		ragMap := make(map[string]*common.AppBrief)
		for _, info := range ragInfos.RagInfos {
			ragMap[info.AppId] = info
		}
		for _, info := range resp.Infos {
			item := response.MyAppItem{
				AppId:       info.AppId,
				AppType:     info.AppType,
				PublishType: info.PublishType,
				CreatedAt:   info.CreatedAt,
			}
			if ragInfo, ok := ragMap[info.AppId]; ok {
				item.Name = ragInfo.Name
				item.Avatar = cacheAppAvatar(ctx, ragInfo.AvatarPath, appType)
			}
			items = append(items, item)
		}

	case constant.AppTypeWorkflow, constant.AppTypeChatflow:
		workflowRet, err := ListWorkflowByIDs(ctx, "", appIds)
		if err != nil {
			log.Errorf("app select get workflow info err: %v", err)
			return nil, err
		}
		workflowMap := make(map[string]*response.CozeWorkflowListDataWorkflow)
		for _, info := range workflowRet.Workflows {
			workflowMap[info.WorkflowId] = info
		}
		for _, info := range resp.Infos {
			item := response.MyAppItem{
				AppId:       info.AppId,
				AppType:     info.AppType,
				PublishType: info.PublishType,
				CreatedAt:   info.CreatedAt,
			}
			if workflowInfo, ok := workflowMap[info.AppId]; ok {
				item.Name = workflowInfo.Name
				item.Avatar = cacheWorkflowAvatar(workflowInfo.URL, appType)
			}
			items = append(items, item)
		}
	}
	return &response.ListResult{
		List:  items,
		Total: int64(len(items)),
	}, nil
}

func getAppDisplayName(displayNameMap map[string]string, appId string) string {
	if displayName, ok := displayNameMap[appId]; ok {
		return displayName
	}
	return "该应用已被删除"
}

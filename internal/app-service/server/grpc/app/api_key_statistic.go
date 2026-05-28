package app

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	app_service "github.com/UnicomAI/wanwu/api/proto/app-service"
	err_code "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/app-service/client/orm"
)

func (s *Service) GetAPIKeyStatistic(ctx context.Context, req *app_service.GetAPIKeyStatisticReq) (*app_service.APIKeyStatistic, error) {
	stats, err := s.cli.GetAPIKeyStatistic(ctx, req.OrgIds, req.UserIds, req.StartDate, req.EndDate, req.ApiKeyIds, req.MethodPaths)
	if err != nil {
		return nil, errStatus(err_code.Code_AppAPIKeyRecord, err)
	}
	return convertAPIKeyStatistic(stats), nil
}

func (s *Service) GetAPIKeyStatisticList(ctx context.Context, req *app_service.GetAPIKeyStatisticListReq) (*app_service.GetAPIKeyStatisticListResp, error) {
	stats, err := s.cli.GetAPIKeyStatisticList(ctx, req.OrgIds, req.UserIds, req.StartDate, req.EndDate, req.ApiKeyIds, req.MethodPaths, toOffset(req), req.PageSize)
	if err != nil {
		return nil, errStatus(err_code.Code_AppAPIKeyRecord, err)
	}
	return convertAPIKeyStatisticList(stats), nil
}

func (s *Service) GetAPIKeyStatisticRecord(ctx context.Context, req *app_service.GetAPIKeyStatisticRecordReq) (*app_service.GetAPIKeyStatisticRecordResp, error) {
	records, err := s.cli.GetAPIKeyStatisticRecord(ctx, req.OrgIds, req.UserIds, req.StartDate, req.EndDate, req.ApiKeyIds, req.MethodPaths, toOffset(req), req.PageSize)
	if err != nil {
		return nil, errStatus(err_code.Code_AppAPIKeyRecord, err)
	}
	return convertAPIKeyStatisticRecordList(records), nil
}

func (s *Service) RecordAPIKeyStatistic(ctx context.Context, req *app_service.RecordAPIKeyStatisticReq) (*emptypb.Empty, error) {
	err := s.cli.RecordAPIKeyStatistic(ctx, req.UserId, req.OrgId, req.ApiKeyId, req.MethodPath,
		req.CallTime, req.HttpStatus, req.IsStream, req.StreamCosts, req.NonStreamCosts, req.RequestBody, req.ResponseBody)
	if err != nil {
		return nil, errStatus(err_code.Code_AppAPIKeyRecord, err)
	}
	return &emptypb.Empty{}, nil
}

func convertAPIKeyStatistic(stats *orm.APIKeyStatistic) *app_service.APIKeyStatistic {
	return &app_service.APIKeyStatistic{
		Overview: convertAPIKeyStatisticOverview(stats.Overview),
		Trend:    convertAPIKeyStatisticTrend(stats.Trend),
	}
}

func convertAPIKeyStatisticOverview(overview orm.APIKeyStatisticOverview) *app_service.APIKeyStatisticOverview {
	return &app_service.APIKeyStatisticOverview{
		CallCount:         convertAPIKeyStatisticOverviewItem(overview.CallCount),
		CallFailure:       convertAPIKeyStatisticOverviewItem(overview.CallFailure),
		AvgStreamCosts:    convertAPIKeyStatisticOverviewItem(overview.AvgStreamCosts),
		AvgNonStreamCosts: convertAPIKeyStatisticOverviewItem(overview.AvgNonStreamCosts),
		StreamCount:       convertAPIKeyStatisticOverviewItem(overview.StreamCount),
		NonStreamCount:    convertAPIKeyStatisticOverviewItem(overview.NonStreamCount),
	}
}

func convertAPIKeyStatisticOverviewItem(item orm.APIKeyStatisticOverviewItem) *app_service.APIKeyStatisticOverviewItem {
	return &app_service.APIKeyStatisticOverviewItem{
		Value:            item.Value,
		PeriodOverPeriod: item.PeriodOverPeriod,
	}
}

func convertAPIKeyStatisticTrend(trend orm.APIKeyStatisticTrend) *app_service.APIKeyStatisticTrend {
	return &app_service.APIKeyStatisticTrend{
		ApiCalls: convertStatisticChart(trend.APICalls),
	}
}

func convertAPIKeyStatisticList(list *orm.APIKeyStatisticList) *app_service.GetAPIKeyStatisticListResp {
	items := make([]*app_service.APIKeyStatisticItem, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, &app_service.APIKeyStatisticItem{
			ApiKeyId:          item.APIKeyID,
			MethodPath:        item.MethodPath,
			OrgId:             item.OrgId,
			UserId:            item.UserId,
			CallCount:         item.CallCount,
			CallFailure:       item.CallFailure,
			AvgStreamCosts:    item.AvgStreamCosts,
			AvgNonStreamCosts: item.AvgNonStreamCosts,
			StreamCount:       item.StreamCount,
			NonStreamCount:    item.NonStreamCount,
		})
	}
	return &app_service.GetAPIKeyStatisticListResp{
		Items: items,
		Total: list.Total,
	}
}

func convertAPIKeyStatisticRecordList(list *orm.APIKeyStatisticRecordList) *app_service.GetAPIKeyStatisticRecordResp {
	items := make([]*app_service.APIKeyStatisticRecordItem, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, &app_service.APIKeyStatisticRecordItem{
			ApiKeyId:       item.APIKeyID,
			MethodPath:     item.MethodPath,
			OrgId:          item.OrgId,
			UserId:         item.UserId,
			CallTime:       item.CallTime,
			ResponseStatus: item.ResponseStatus,
			StreamCosts:    item.StreamCosts,
			NonStreamCosts: item.NonStreamCosts,
			RequestBody:    item.RequestBody,
			ResponseBody:   item.ResponseBody,
		})
	}
	return &app_service.GetAPIKeyStatisticRecordResp{
		Items: items,
		Total: list.Total,
	}
}

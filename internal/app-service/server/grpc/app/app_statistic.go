package app

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	app_service "github.com/UnicomAI/wanwu/api/proto/app-service"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/app-service/client/orm"
)

func (s *Service) GetAppStatistic(ctx context.Context, req *app_service.GetAppStatisticReq) (*app_service.AppStatistic, error) {
	stats, err := s.cli.GetAppStatistic(ctx, req.OrgIds, req.UserIds, req.StartDate, req.EndDate, req.AppIds, req.AppType)
	if err != nil {
		return nil, errStatus(errs.Code_AppModelRecord, err)
	}
	return convertAppStatistic(stats), nil
}

func (s *Service) GetAppStatisticList(ctx context.Context, req *app_service.GetAppStatisticListReq) (*app_service.GetAppStatisticListResp, error) {
	stats, err := s.cli.GetAppStatisticList(ctx, req.OrgIds, req.UserIds, req.StartDate, req.EndDate, req.AppIds, req.AppType, toOffset(req), req.PageSize)
	if err != nil {
		return nil, errStatus(errs.Code_AppModelRecord, err)
	}
	return convertAppStatisticList(stats), nil
}

func (s *Service) RecordAppStatistic(ctx context.Context, req *app_service.RecordAppStatisticReq) (*emptypb.Empty, error) {
	err := s.cli.RecordAppStatistic(ctx, req.UserId, req.OrgId, req.AppId, req.AppType,
		req.IsSuccess, req.IsStream, req.StreamCosts, req.NonStreamCosts, req.Source)
	if err != nil {
		return nil, errStatus(errs.Code_AppModelRecord, err)
	}
	return &emptypb.Empty{}, nil
}

func convertAppStatistic(stats *orm.AppStatistic) *app_service.AppStatistic {
	return &app_service.AppStatistic{
		Overview: convertAppStatisticOverview(stats.Overview),
		Trend:    convertAppStatisticTrend(stats.Trend),
	}
}

func convertAppStatisticOverview(overview orm.AppStatisticOverview) *app_service.AppStatisticOverview {
	return &app_service.AppStatisticOverview{
		CallCount:         convertStatisticOverviewItem(overview.CallCount),
		CallFailure:       convertStatisticOverviewItem(overview.CallFailure),
		StreamCount:       convertStatisticOverviewItem(overview.StreamCount),
		NonStreamCount:    convertStatisticOverviewItem(overview.NonStreamCount),
		AvgStreamCosts:    convertStatisticOverviewItem(overview.AvgStreamCosts),
		AvgNonStreamCosts: convertStatisticOverviewItem(overview.AvgNonStreamCosts),
	}
}

func convertStatisticOverviewItem(item orm.StatisticOverviewItem) *app_service.ModelStatisticOverviewItem {
	return &app_service.ModelStatisticOverviewItem{
		Value:            item.Value,
		PeriodOverPeriod: item.PeriodOverPeriod,
	}
}

func convertAppStatisticTrend(trend orm.AppStatisticTrend) *app_service.AppStatisticTrend {
	return &app_service.AppStatisticTrend{
		CallTrend: convertStatisticChart(trend.CallTrend),
	}
}

func convertAppStatisticList(list *orm.AppStatisticList) *app_service.GetAppStatisticListResp {
	items := make([]*app_service.AppStatisticItem, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, &app_service.AppStatisticItem{
			AppId:             item.AppId,
			AppType:           item.AppType,
			OrgId:             item.OrgId,
			UserId:            item.UserId,
			CallCount:         item.CallCount,
			CallFailure:       item.CallFailure,
			FailureRate:       item.FailureRate,
			StreamCount:       item.StreamCount,
			NonStreamCount:    item.NonStreamCount,
			AvgStreamCosts:    item.AvgStreamCosts,
			AvgNonStreamCosts: item.AvgNonStreamCosts,
		})
	}
	return &app_service.GetAppStatisticListResp{
		Items: items,
		Total: list.Total,
	}
}

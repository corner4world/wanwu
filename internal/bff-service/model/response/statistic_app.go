package response

import "github.com/UnicomAI/wanwu/internal/bff-service/model/request"

type AppStatistic struct {
	Overview AppStatisticOverview `json:"overview"`
	Trend    AppStatisticTrend    `json:"trend"`
}

type AppStatisticOverview struct {
	CallCount         StatisticOverviewItem `json:"callCount"`
	CallFailure       StatisticOverviewItem `json:"callFailure"`
	StreamCount       StatisticOverviewItem `json:"streamCount"`
	NonStreamCount    StatisticOverviewItem `json:"nonStreamCount"`
	AvgStreamCosts    StatisticOverviewItem `json:"avgStreamCosts"`
	AvgNonStreamCosts StatisticOverviewItem `json:"avgNonStreamCosts"`
}

type AppStatisticTrend struct {
	CallTrend StatisticChart `json:"callTrend"`
}

type AppStatisticItem struct {
	AppId             string  `json:"appId"`
	AppType           string  `json:"appType"`
	AppName           string  `json:"appName"`
	UserName          string  `json:"userName"`
	OrgName           string  `json:"orgName"`
	CallCount         int32   `json:"callCount"`
	CallFailure       int32   `json:"callFailure"`
	FailureRate       float32 `json:"failureRate"`
	AvgStreamCosts    float32 `json:"avgStreamCosts"`
	AvgNonStreamCosts float32 `json:"avgNonStreamCosts"`
	StreamCount       int32   `json:"streamCount"`
	NonStreamCount    int32   `json:"nonStreamCount"`
}

type MyAppItem struct {
	AppId       string         `json:"appId"`
	Name        string         `json:"name"`
	AppType     string         `json:"appType"`
	Avatar      request.Avatar `json:"avatar"` // 图标
	PublishType string         `json:"publishType"`
	CreatedAt   int64          `json:"createdAt"`
}

package orm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/app-service/client/model"
	"github.com/UnicomAI/wanwu/internal/app-service/client/orm/sqlopt"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/redis"
	"github.com/UnicomAI/wanwu/pkg/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Lua script for atomic Redis updates of APIKey statistics
const luaUpdateAPIKeyStats = `
local key = KEYS[1]
local field = KEYS[2]
local delta = cjson.decode(ARGV[1])
local expire = tonumber(ARGV[2])

local current = redis.call('HGET', key, field)

if current then
    local record = cjson.decode(current)
    record.callCount = record.callCount + delta.callCount
    record.callFailure = record.callFailure + delta.callFailure
    record.streamCount = record.streamCount + delta.streamCount
    record.nonStreamCount = record.nonStreamCount + delta.nonStreamCount
    record.streamFailure = record.streamFailure + delta.streamFailure
    record.nonStreamFailure = record.nonStreamFailure + delta.nonStreamFailure
    record.streamCosts = record.streamCosts + delta.streamCosts
    record.nonStreamCosts = record.nonStreamCosts + delta.nonStreamCosts
    redis.call('HSET', key, field, cjson.encode(record))
else
    redis.call('HSET', key, field, ARGV[1])
end

redis.call('EXPIRE', key, expire)
return 1
`

// Redis record structure for API Key statistics (stored per day, per API Key + user/org + methodPath)
type APIKeyRecordStats struct {
	APIKeyID         string `json:"apiKeyId"`
	MethodPath       string `json:"methodPath"`
	CallCount        int32  `json:"callCount"`
	CallFailure      int32  `json:"callFailure"`
	StreamCount      int32  `json:"streamCount"`
	NonStreamCount   int32  `json:"nonStreamCount"`
	StreamFailure    int32  `json:"streamFailure"`
	NonStreamFailure int32  `json:"nonStreamFailure"`
	StreamCosts      int64  `json:"streamCosts"`
	NonStreamCosts   int64  `json:"nonStreamCosts"`
}

// APIKeyStatistic wraps the overview and trend for API Key statistics
type APIKeyStatistic struct {
	Overview APIKeyStatisticOverview `json:"overview"`
	Trend    APIKeyStatisticTrend    `json:"trend"`
}

type APIKeyStatisticOverview struct {
	CallCount         APIKeyStatisticOverviewItem `json:"callCount"`
	CallFailure       APIKeyStatisticOverviewItem `json:"callFailure"`
	AvgStreamCosts    APIKeyStatisticOverviewItem `json:"avgStreamCosts"`
	AvgNonStreamCosts APIKeyStatisticOverviewItem `json:"avgNonStreamCosts"`
	StreamCount       APIKeyStatisticOverviewItem `json:"streamCount"`
	NonStreamCount    APIKeyStatisticOverviewItem `json:"nonStreamCount"`
}

type APIKeyStatisticOverviewItem struct {
	Value float32 `json:"value"`
	// PeriodOverPeriod tracks the value comparison with the previous period
	// (capital P to match protobuf field name PeriodOverPeriod).
	PeriodOverPeriod float32 `json:"periodOverPeriod"`
}

type APIKeyStatisticTrend struct {
	APICalls StatisticChart `json:"apiCalls"`
}

type APIKeyStatisticItem struct {
	APIKeyID          string
	MethodPath        string
	OrgId             string
	UserId            string
	CallCount         int32
	CallFailure       int32
	AvgStreamCosts    float32
	AvgNonStreamCosts float32
	StreamCount       int32
	NonStreamCount    int32
}

type APIKeyStatisticRecordItem struct {
	APIKeyID       string
	MethodPath     string
	OrgId          string
	UserId         string
	CallTime       int64
	ResponseStatus string
	StreamCosts    int64
	NonStreamCosts int64
	RequestBody    string
	ResponseBody   string
}

type APIKeyStatisticList struct {
	Items []APIKeyStatisticItem
	Total int32
}

type APIKeyStatisticRecordList struct {
	Items []APIKeyStatisticRecordItem
	Total int32
}

// getRedisAPIKeyStatsKey 获取Redis统计key
func getRedisAPIKeyStatsKey(date string) string {
	return fmt.Sprintf("apikeystatistic|%s", date)
}

// getRedisAPIKeyStatsItemField 获取Redis统计字段
func getRedisAPIKeyStatsItemField(apiKeyId, userId, orgId, methodPath string) string {
	return fmt.Sprintf("%s|%s|%s|%s", apiKeyId, userId, orgId, methodPath)
}

// parseRedisAPIKeyStatsItem 解析Redis字段
func parseRedisAPIKeyStatsItem(field string) (string, string, string, string, bool) {
	parts := strings.Split(field, "|")
	if len(parts) != 4 {
		return "", "", "", "", false
	}
	return parts[0], parts[1], parts[2], parts[3], true
}

// getRedisAPIKeyStatsItemValue 解析Redis value
func getRedisAPIKeyStatsItemValue(value string) (*APIKeyRecordStats, error) {
	record := &APIKeyRecordStats{}
	if err := json.Unmarshal([]byte(value), record); err != nil {
		return nil, fmt.Errorf("unmarshal value %v err: %v", value, err)
	}
	return record, nil
}

// RecordAPIKeyStatistic 记录 API Key 统计（调用 Redis 并写入明细）
func (c *Client) RecordAPIKeyStatistic(ctx context.Context, userId, orgId, apiKeyId, methodPath string, callTime int64, httpStatus string, isStream bool, streamCosts, nonStreamCosts int64, requestBody, responseBody string) *errs.Status {
	if userId == "" || apiKeyId == "" {
		return toErrStatus("app_api_key_statistic_record", fmt.Sprintf("invalid parameters: userId=%s apiKeyId=%s", userId, apiKeyId))
	}
	// today
	today := util.Time2Date(time.Now().UnixMilli())
	// Determine success based on HTTP status code
	isSuccess := httpStatus == "200"
	// build delta
	record := APIKeyRecordStats{
		APIKeyID:         apiKeyId,
		MethodPath:       methodPath,
		CallCount:        1,
		CallFailure:      0,
		StreamCount:      0,
		NonStreamCount:   0,
		StreamFailure:    0,
		NonStreamFailure: 0,
		StreamCosts:      streamCosts,
		NonStreamCosts:   nonStreamCosts,
	}
	if isStream {
		record.StreamCount = 1
		if !isSuccess {
			record.StreamFailure = 1
		}
	} else {
		record.NonStreamCount = 1
		if !isSuccess {
			record.NonStreamFailure = 1
		}
	}
	if !isSuccess {
		record.CallFailure = 1
	}

	// Redis update
	key := getRedisAPIKeyStatsKey(today)
	field := getRedisAPIKeyStatsItemField(apiKeyId, userId, orgId, methodPath)

	deltaJSON, _ := json.Marshal(record)
	if _, err := redis.App().Eval(ctx, luaUpdateAPIKeyStats, []string{key, field}, string(deltaJSON), 30*24*3600); err != nil {
		log.Errorf("redis api key stat update err: %v", err)
	}

	// Persist a detailed call record
	rec := &model.APIKeyRecord{
		OrgID:          orgId,
		UserID:         userId,
		APIKeyID:       apiKeyId,
		MethodPath:     methodPath,
		CallTime:       callTime,
		ResponseStatus: httpStatus,
		IsStream:       isStream,
		StreamCosts:    streamCosts,
		NonStreamCosts: nonStreamCosts,
		Date:           today,
		RequestBody:    requestBody,
		ResponseBody:   responseBody,
	}
	if err := c.db.WithContext(ctx).Create(rec).Error; err != nil {
		log.Errorf("failed to save api key call record: %v", err)
	}
	return nil
}

// GetAPIKeyStatistic 获取 API Key 统计（概览+趋势）
func (c *Client) GetAPIKeyStatistic(ctx context.Context, orgIds, userIds []string, startDate, endDate string, apiKeyIds []string, methodPaths []string) (*APIKeyStatistic, *errs.Status) {
	// quick guard
	if startDate > endDate {
		return nil, toErrStatus("app_api_key_statistic_get", fmt.Errorf("startDate %v greater than endDate %v", startDate, endDate).Error())
	}
	today := util.Time2Date(time.Now().UnixMilli())
	if err := updateAPIKeyStats(ctx, today, c.db); err != nil {
		log.Errorf("sync api key stats for today %v err: %v", today, err)
	}

	// overview and trend (real aggregations)
	overview, err := statisticAPIKeyStatsOverview(ctx, c.db, orgIds, userIds, startDate, endDate, apiKeyIds, methodPaths)
	if err != nil {
		return nil, toErrStatus("app_api_key_statistic_get", err.Error())
	}
	trend, err := statisticAPIKeyStatsTrend(ctx, c.db, orgIds, userIds, startDate, endDate, apiKeyIds, methodPaths)
	if err != nil {
		return nil, toErrStatus("app_api_key_statistic_get", err.Error())
	}
	return &APIKeyStatistic{Overview: *overview, Trend: *trend}, nil
}

// GetAPIKeyStatisticList 获取聚合统计列表（分页）
func (c *Client) GetAPIKeyStatisticList(ctx context.Context, orgIds, userIds []string, startDate, endDate string, apiKeyIds []string, methodPaths []string, offset, limit int32) (*APIKeyStatisticList, *errs.Status) {
	if startDate > endDate {
		return nil, toErrStatus("app_api_key_statistic_list", fmt.Errorf("startDate %v greater than endDate %v", startDate, endDate).Error())
	}

	today := util.Time2Date(time.Now().UnixMilli())
	if err := updateAPIKeyStats(ctx, today, c.db); err != nil {
		log.Errorf("sync api key stats for today %v err: %v", today, err)
	}

	items, total, err := getAPIKeyStatisticList(ctx, c.db, orgIds, userIds, startDate, endDate, apiKeyIds, methodPaths, offset, limit)
	if err != nil {
		return nil, toErrStatus("app_api_key_statistic_list", err.Error())
	}
	return &APIKeyStatisticList{Items: items, Total: total}, nil
}

// GetAPIKeyStatisticRecord 获取 API Key 调用明细（分页）
func (c *Client) GetAPIKeyStatisticRecord(ctx context.Context, orgIds, userIds []string, startDate, endDate string, apiKeyIds, methodPaths []string, offset, limit int32) (*APIKeyStatisticRecordList, *errs.Status) {
	items, total, err := getAPIKeyStatisticRecordList(ctx, c.db, orgIds, userIds, startDate, endDate, apiKeyIds, methodPaths, offset, limit)
	if err != nil {
		return nil, toErrStatus("app_api_key_statistic_record_list", err.Error())
	}
	return &APIKeyStatisticRecordList{Items: items, Total: total}, nil
}

// --- internal helpers ---

// updateAPIKeyStats 将 Redis 中的日累积数据刷新到数据库表 api_key_statistic
func updateAPIKeyStats(ctx context.Context, date string, db *gorm.DB) error {
	key := getRedisAPIKeyStatsKey(date)
	resultMap, err := redis.App().HGetAll(ctx, key)
	if err != nil {
		return fmt.Errorf("redis HGetAll key %v failed: %v", key, err)
	}
	if len(resultMap) == 0 {
		return nil
	}
	for _, item := range resultMap {
		if apiKeyId, userId, orgId, methodPath, ok := parseRedisAPIKeyStatsItem(item.K); ok {
			record, err := getRedisAPIKeyStatsItemValue(item.V)
			if err != nil {
				log.Errorf("get api key stat item %v err: %v", item.K, err)
				continue
			}
			if err := updateAPIKeyStatsByRecord(ctx, db, apiKeyId, userId, orgId, methodPath, date, record); err != nil {
				log.Errorf("update date %v api key stat failed for apiKeyId %v userId %v orgId %v err: %v", date, apiKeyId, userId, orgId, err)
			}
		}
	}
	return nil
}

func updateAPIKeyStatsByRecord(ctx context.Context, db *gorm.DB, apiKeyId, userId, orgId, methodPath, date string, record *APIKeyRecordStats) error {
	stat := &model.APIKeyStatistic{
		OrgID:            orgId,
		UserID:           userId,
		APIKeyID:         apiKeyId,
		MethodPath:       methodPath,
		Date:             date,
		CallCount:        record.CallCount,
		CallFailure:      record.CallFailure,
		StreamCount:      record.StreamCount,
		NonStreamCount:   record.NonStreamCount,
		StreamFailure:    record.StreamFailure,
		NonStreamFailure: record.NonStreamFailure,
		StreamCosts:      record.StreamCosts,
		NonStreamCosts:   record.NonStreamCosts,
	}

	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "org_id"},
			{Name: "user_id"},
			{Name: "api_key_id"},
			{Name: "method_path"},
			{Name: "date"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"call_count", "call_failure", "stream_count",
			"non_stream_count", "stream_failure", "non_stream_failure",
			"stream_costs", "non_stream_costs",
		}),
	}).Create(stat).Error
}

// statisticAPIKeyStatsOverview 统计API Key概览数据（占位实现，可扩展）
func statisticAPIKeyStatsOverview(ctx context.Context, db *gorm.DB, orgIds, userIds []string, startDate, endDate string, apiKeyIds []string, methodPaths []string) (*APIKeyStatisticOverview, error) {
	prevPeriod, currPeriod, err := util.PreviousDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}
	current, err := apiKeyStatsByDateRange(ctx, db, orgIds, userIds, currPeriod, apiKeyIds, methodPaths)
	if err != nil {
		return nil, err
	}
	previous, err := apiKeyStatsByDateRange(ctx, db, orgIds, userIds, prevPeriod, apiKeyIds, methodPaths)
	if err != nil {
		return nil, err
	}
	current.CallCount.PeriodOverPeriod = calculatePoP(current.CallCount.Value, previous.CallCount.Value)
	current.CallFailure.PeriodOverPeriod = calculatePoP(current.CallFailure.Value, previous.CallFailure.Value)
	current.StreamCount.PeriodOverPeriod = calculatePoP(current.StreamCount.Value, previous.StreamCount.Value)
	current.NonStreamCount.PeriodOverPeriod = calculatePoP(current.NonStreamCount.Value, previous.NonStreamCount.Value)
	current.AvgStreamCosts.PeriodOverPeriod = calculatePoP(current.AvgStreamCosts.Value, previous.AvgStreamCosts.Value)
	current.AvgNonStreamCosts.PeriodOverPeriod = calculatePoP(current.AvgNonStreamCosts.Value, previous.AvgNonStreamCosts.Value)
	return current, nil
}

// apiKeyStatsByDateRange 按日期范围获取API Key统计数据（占位实现）
func apiKeyStatsByDateRange(ctx context.Context, db *gorm.DB, orgIds, userIds []string, dates []string, apiKeyIds []string, methodPaths []string) (*APIKeyStatisticOverview, error) {
	if len(dates) == 0 {
		return &APIKeyStatisticOverview{}, nil
	}
	// dates[0] is start, dates[len-1] is end
	startDate := dates[0]
	endDate := dates[len(dates)-1]
	type sumRow struct {
		CallCount        int64
		CallFailure      int64
		StreamCount      int64
		NonStreamCount   int64
		StreamFailure    int64
		NonStreamFailure int64
		StreamCosts      int64
		NonStreamCosts   int64
	}
	var row sumRow
	opts := []sqlopt.SQLOption{
		sqlopt.StartDate(startDate),
		sqlopt.EndDate(endDate),
		sqlopt.WithAPIKeyIds(apiKeyIds),
		sqlopt.WithMethodPaths(methodPaths),
		sqlopt.WithOrgIDs(orgIds),
		sqlopt.WithUserIDs(userIds),
	}
	if err := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).Model(&model.APIKeyStatistic{}).
		Select("SUM(call_count) as CallCount, SUM(call_failure) as CallFailure, SUM(stream_count) as StreamCount, SUM(non_stream_count) as NonStreamCount, SUM(stream_failure) as StreamFailure, SUM(non_stream_failure) as NonStreamFailure, SUM(stream_costs) as StreamCosts, SUM(non_stream_costs) as NonStreamCosts").First(&row).Error; err != nil {
		return nil, fmt.Errorf("api key stat date range err: %v", err)
	}
	// Compute averages for the overview fields
	var avgStreamCosts float32 = 0
	var avgNonStreamCosts float32 = 0
	if row.StreamCount > 0 {
		avgStreamCosts = float32(row.StreamCosts) / float32(row.StreamCount)
	}
	if row.NonStreamCount > 0 {
		avgNonStreamCosts = float32(row.NonStreamCosts) / float32(row.NonStreamCount)
	}
	overview := &APIKeyStatisticOverview{
		CallCount:         APIKeyStatisticOverviewItem{Value: float32(row.CallCount)},
		CallFailure:       APIKeyStatisticOverviewItem{Value: float32(row.CallFailure)},
		AvgStreamCosts:    APIKeyStatisticOverviewItem{Value: avgStreamCosts},
		AvgNonStreamCosts: APIKeyStatisticOverviewItem{Value: avgNonStreamCosts},
		StreamCount:       APIKeyStatisticOverviewItem{Value: float32(row.StreamCount)},
		NonStreamCount:    APIKeyStatisticOverviewItem{Value: float32(row.NonStreamCount)},
	}
	return overview, nil
}

// statisticAPIKeyStatsTrend 统计API Key趋势数据
func statisticAPIKeyStatsTrend(ctx context.Context, db *gorm.DB, orgIds, userIds []string, startDate, endDate string, apiKeyIds []string, methodPaths []string) (*APIKeyStatisticTrend, error) {
	dates, err := buildDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	var stats []model.APIKeyStatistic
	opts := []sqlopt.SQLOption{
		sqlopt.StartDate(startDate),
		sqlopt.EndDate(endDate),
		sqlopt.WithAPIKeyIds(apiKeyIds),
		sqlopt.WithMethodPaths(methodPaths),
		sqlopt.WithOrgIDs(orgIds),
		sqlopt.WithUserIDs(userIds),
	}
	if err := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).Model(&model.APIKeyStatistic{}).
		Select("date, SUM(call_count) as call_count, SUM(call_failure) as call_failure").
		Group("date").Order("date").Find(&stats).Error; err != nil {
		return nil, fmt.Errorf("api key stat trend err: %v", err)
	}

	lineNames := []string{
		"app_statistic_api_call_count_total",
		"app_statistic_api_call_success",
		"app_statistic_api_call_failure",
	}

	lines := buildChartLines(stats, dates,
		func(r model.APIKeyStatistic) string { return r.Date },
		func(r model.APIKeyStatistic) map[string]float32 {
			return map[string]float32{
				"app_statistic_api_call_count_total": float32(r.CallCount),
				"app_statistic_api_call_success":     float32(r.CallCount - r.CallFailure),
				"app_statistic_api_call_failure":     float32(r.CallFailure),
			}
		},
		lineNames,
	)

	return &APIKeyStatisticTrend{
		APICalls: StatisticChart{
			Name:  "app_statistic_api_key_call_trend",
			Lines: lines,
		},
	}, nil
}

// getAPIKeyStatisticList 辅助查询（占位实现）
func getAPIKeyStatisticList(ctx context.Context, db *gorm.DB, orgIds, userIds []string, startDate, endDate string, apiKeyIds []string, methodPaths []string, offset, limit int32) ([]APIKeyStatisticItem, int32, error) {
	opts := []sqlopt.SQLOption{
		sqlopt.StartDate(startDate),
		sqlopt.EndDate(endDate),
		sqlopt.WithAPIKeyIds(apiKeyIds),
		sqlopt.WithMethodPaths(methodPaths),
		sqlopt.WithOrgIDs(orgIds),
		sqlopt.WithUserIDs(userIds),
	}
	var total int64
	countQuery := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).Model(&model.APIKeyStatistic{}).
		Select("COUNT(DISTINCT api_key_id,method_path)")
	countQuery.Count(&total)
	var stats []model.APIKeyStatistic
	query := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).
		Select("api_key_id, ANY_VALUE(org_id) as org_id, ANY_VALUE(user_id) as user_id, ANY_VALUE(method_path) as method_path, SUM(call_count) as call_count, SUM(call_failure) as call_failure, SUM(stream_count) as stream_count, SUM(non_stream_count) as non_stream_count, SUM(stream_failure) as stream_failure, SUM(non_stream_failure) as non_stream_failure, SUM(stream_costs) as stream_costs, SUM(non_stream_costs) as non_stream_costs").
		Group("api_key_id, method_path").Order("call_count DESC").Offset(int(offset)).Limit(int(limit))

	if err := query.Find(&stats).Error; err != nil {
		return nil, 0, fmt.Errorf("get api key statistic list err: %v", err)
	}

	items := make([]APIKeyStatisticItem, 0, len(stats))
	for _, s := range stats {
		callCount := s.CallCount
		callFailure := s.CallFailure
		avgStreamCosts := float32(0)
		avgNonStreamCosts := float32(0)
		if s.StreamCount > 0 {
			avgStreamCosts = float32(s.StreamCosts) / float32(s.StreamCount)
		}
		if s.NonStreamCount > 0 {
			avgNonStreamCosts = float32(s.NonStreamCosts) / float32(s.NonStreamCount)
		}
		items = append(items, APIKeyStatisticItem{
			APIKeyID:          s.APIKeyID,
			MethodPath:        s.MethodPath,
			OrgId:             s.OrgID,
			UserId:            s.UserID,
			CallCount:         int32(callCount),
			CallFailure:       int32(callFailure),
			AvgStreamCosts:    avgStreamCosts,
			AvgNonStreamCosts: avgNonStreamCosts,
			StreamCount:       int32(s.StreamCount),
			NonStreamCount:    int32(s.NonStreamCount),
		})
	}

	return items, int32(total), nil
}

// getAPIKeyStatisticRecordList 辅助查询（占位实现）
func getAPIKeyStatisticRecordList(ctx context.Context, db *gorm.DB, orgIds, userIds []string, startDate, endDate string, apiKeyIds, methodPaths []string, offset, limit int32) ([]APIKeyStatisticRecordItem, int32, error) {
	var total int64
	var records []model.APIKeyRecord
	opts := []sqlopt.SQLOption{
		sqlopt.StartDate(startDate),
		sqlopt.EndDate(endDate),
		sqlopt.WithOrgIDs(orgIds),
		sqlopt.WithUserIDs(userIds),
	}
	if len(apiKeyIds) > 0 {
		opts = append(opts, sqlopt.WithAPIKeyIds(apiKeyIds))
	}
	if len(methodPaths) > 0 {
		opts = append(opts, sqlopt.WithMethodPaths(methodPaths))
	}
	if err := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).Model(&model.APIKeyRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := sqlopt.SQLOptions(opts...).Apply(db).WithContext(ctx).Model(&model.APIKeyRecord{}).Order("call_time DESC").Offset(int(offset)).Limit(int(limit)).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	items := make([]APIKeyStatisticRecordItem, 0, len(records))
	for _, r := range records {
		items = append(items, APIKeyStatisticRecordItem{
			APIKeyID:       r.APIKeyID,
			MethodPath:     r.MethodPath,
			OrgId:          r.OrgID,
			UserId:         r.UserID,
			CallTime:       r.CallTime,
			ResponseStatus: r.ResponseStatus,
			StreamCosts:    r.StreamCosts,
			NonStreamCosts: r.NonStreamCosts,
			RequestBody:    r.RequestBody,
			ResponseBody:   r.ResponseBody,
		})
	}
	return items, int32(total), nil
}

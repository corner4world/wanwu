package response

type APIKeyStatistic struct {
	Overview APIKeyStatisticOverview `json:"overview"`
	Trend    APIKeyStatisticTrend    `json:"trend"`
}

type APIKeyStatisticOverview struct {
	CallCount         StatisticOverviewItem `json:"callCount"`
	CallFailure       StatisticOverviewItem `json:"callFailure"`
	AvgStreamCosts    StatisticOverviewItem `json:"avgStreamCosts"`
	AvgNonStreamCosts StatisticOverviewItem `json:"avgNonStreamCosts"`
	StreamCount       StatisticOverviewItem `json:"streamCount"`
	NonStreamCount    StatisticOverviewItem `json:"nonStreamCount"`
}

type APIKeyStatisticTrend struct {
	APICalls StatisticChart `json:"apiCalls"`
}

type APIKeyStatisticItem struct {
	OrgName           string  `json:"orgName"`
	UserName          string  `json:"userName"`
	Name              string  `json:"name"`
	APIKey            string  `json:"apiKey"`
	MethodPath        string  `json:"methodPath"`
	CallCount         int32   `json:"callCount"`
	CallFailure       int32   `json:"callFailure"`
	AvgStreamCosts    float32 `json:"avgStreamCosts"`
	AvgNonStreamCosts float32 `json:"avgNonStreamCosts"`
	StreamCount       int32   `json:"streamCount"`
	NonStreamCount    int32   `json:"nonStreamCount"`
}

type APIKeyStatisticRecordItem struct {
	OrgName        string `json:"orgName"`
	UserName       string `json:"userName"`
	Name           string `json:"name"`
	APIKey         string `json:"apiKey"`
	MethodPath     string `json:"methodPath"`
	CallTime       string `json:"callTime"`
	ResponseStatus string `json:"responseStatus"`
	StreamCosts    int64  `json:"streamCosts"`
	NonStreamCosts int64  `json:"nonStreamCosts"`
	RequestBody    string `json:"requestBody"`
	ResponseBody   string `json:"responseBody"`
}

type ApiKeyStatisticRouteItem struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

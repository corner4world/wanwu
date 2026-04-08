package service

import (
	"encoding/json"
	"os"

	ag_ui_util "github.com/UnicomAI/wanwu/pkg/ag-ui-util"
)

var tavilyIconPath = os.Getenv("WANWU_EXTERNAL_ENDPOINT") + "/user/api/v1/cache/mcp/tool/tavilysearch/logo.png"

func WgaFormatBochaWebSearchResult(result string) string {
	var rawResponse struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			QueryContext struct {
				OriginalQuery string `json:"originalQuery"`
			} `json:"queryContext"`
			WebPages struct {
				Value []struct {
					Name     string `json:"name"`
					URL      string `json:"url"`
					Snippet  string `json:"snippet"`
					Summary  string `json:"summary"`
					SiteName string `json:"siteName"`
					SiteIcon string `json:"siteIcon"`
				} `json:"value"`
			} `json:"webPages"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(result), &rawResponse); err != nil {
		return result
	}

	formattedResult := ag_ui_util.WebSearchResult{
		Query:    rawResponse.Data.QueryContext.OriginalQuery,
		WebCount: len(rawResponse.Data.WebPages.Value),
		WebPages: make([]ag_ui_util.WebPage, 0, len(rawResponse.Data.WebPages.Value)),
	}

	for _, page := range rawResponse.Data.WebPages.Value {
		summary := page.Summary
		if summary == "" {
			summary = page.Snippet
		}

		formattedResult.WebPages = append(formattedResult.WebPages, ag_ui_util.WebPage{
			Title:    page.Name,
			SiteName: page.SiteName,
			Icon:     page.SiteIcon,
			Summary:  summary,
			URL:      page.URL,
		})
	}

	formattedJSON, err := json.Marshal(formattedResult)
	if err != nil {
		return result
	}

	return string(formattedJSON)
}

func WgaFormatTavilySearchResult(result string) string {
	var rawResponse struct {
		Query        string  `json:"query"`
		Answer       string  `json:"answer"`
		ResponseTime float64 `json:"response_time"`
		Results      []struct {
			Title         string  `json:"title"`
			URL           string  `json:"url"`
			Content       string  `json:"content"`
			Score         float64 `json:"score"`
			RawContent    string  `json:"raw_content"`
			PublishedDate string  `json:"published_date"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(result), &rawResponse); err != nil {
		return result
	}

	formattedResult := ag_ui_util.WebSearchResult{
		Query:    rawResponse.Query,
		WebCount: len(rawResponse.Results),
		WebPages: make([]ag_ui_util.WebPage, 0, len(rawResponse.Results)),
	}

	for _, page := range rawResponse.Results {
		summary := page.Content
		if summary == "" {
			summary = page.RawContent
		}

		formattedResult.WebPages = append(formattedResult.WebPages, ag_ui_util.WebPage{
			Title:    page.Title,
			SiteName: "Tavily",
			Icon:     tavilyIconPath,
			Summary:  summary,
			URL:      page.URL,
		})
	}

	formattedJSON, err := json.Marshal(formattedResult)
	if err != nil {
		return result
	}

	return string(formattedJSON)
}

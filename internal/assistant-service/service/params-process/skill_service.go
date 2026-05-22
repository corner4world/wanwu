package params_process

import (
	"context"
	"encoding/json"
	"errors"
	net_url "net/url"
	"time"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	"github.com/UnicomAI/wanwu/internal/assistant-service/client/model"
	"github.com/UnicomAI/wanwu/internal/assistant-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/pkg/constant"
	http_client "github.com/UnicomAI/wanwu/pkg/http-client"
	"github.com/UnicomAI/wanwu/pkg/log"
)

type SkillProcess struct {
}

type BuiltinSkillIdListParams struct {
	SkillIdList []string `json:"skillIdList"`
}

type SkillDetailListResp struct {
	Code int64                  `json:"code"`
	Msg  string                 `json:"msg"`
	Data *SkillDetailListResult `json:"data"`
}

type SkillDetailListResult struct {
	SkillList []*SkillDetail `json:"skillList"`
}

type CustomSkillDetailListResp struct {
	Code int64                        `json:"code"`
	Msg  string                       `json:"msg"`
	Data *CustomSkillDetailListResult `json:"data"`
}

type CustomSkillDetailListResult struct {
	SkillList []*CustomSkillDetail `json:"skillList"`
}

type AcquiredSkillDetailListResp struct {
	Code int64                          `json:"code"`
	Msg  string                         `json:"msg"`
	Data *AcquiredSkillDetailListResult `json:"data"`
}

type AcquiredSkillDetailListResult struct {
	SkillList []*AcquiredSkillDetail `json:"skillList"`
}

type SkillDetail struct {
	SkillId       string         `json:"skillId"`             // 模板ID
	Name          string         `json:"name"`                // 模板名称
	Avatar        request.Avatar `json:"avatar"`              // 模板头像
	Author        string         `json:"author"`              // 作者
	Desc          string         `json:"desc"`                // 模板描述
	SkillMarkdown string         `json:"skillMarkdown"`       // 模板markdown预览
	SkillPath     string         `json:"skillPath,omitempty"` // markdown地址，内部使用，不要对外
}

type CustomSkillDetail struct {
	SkillId       string         `json:"skillId"`
	Name          string         `json:"name"`
	Avatar        request.Avatar `json:"avatar"`
	Author        string         `json:"author"`
	Desc          string         `json:"desc"`
	SkillMarkdown string         `json:"skillMarkdown,omitempty"`
	ObjectPath    string         `json:"objectPath,omitempty"`
}

type AcquiredSkillDetail struct {
	SkillId    string         `json:"skillId"`
	Name       string         `json:"name"`
	Avatar     request.Avatar `json:"avatar"`
	Author     string         `json:"author"`
	Desc       string         `json:"desc"`
	ObjectPath string         `json:"objectPath"`
}

func init() {
	AddServiceContainer(&SkillProcess{})
}

func (k *SkillProcess) ServiceType() ServiceType {
	return SkillType
}

func (k *SkillProcess) Prepare(agent *AgentInfo, prepareParams *AgentPrepareParams, clientInfo *ClientInfo, userQueryParams *UserQueryParams) error {
	skills := buildAssistantSkills(agent, clientInfo)
	if len(skills) == 0 {
		return nil
	}

	var builtinSkillIds []string
	var customSkillIds []string
	var acquiredSkillIds []string
	for _, skill := range skills {
		if !skill.Enable {
			continue
		}
		switch skill.SkillType {
		case constant.SkillTypeBuiltIn:
			builtinSkillIds = append(builtinSkillIds, skill.SkillId)
		case constant.SkillTypeCustom:
			customSkillIds = append(customSkillIds, skill.SkillId)
		case constant.SkillTypeAcquired:
			acquiredSkillIds = append(acquiredSkillIds, skill.SkillId)
		}
	}

	//获取custom skill详情
	if len(customSkillIds) > 0 {
		customSkillResp, err := SearchCustomSkillList(context.Background(), &BuiltinSkillIdListParams{SkillIdList: customSkillIds})
		if err != nil {
			log.Errorf("Assistant服务获取Custom Skill详情失败，assistantId: %d, error: %v", agent.Assistant.ID, err)
			return err
		}
		if customSkillResp.Data != nil {
			prepareParams.CustomSkillList = customSkillResp.Data.SkillList
		}
	}

	//获取acquired skill详情
	if len(acquiredSkillIds) > 0 {
		acquiredSkillResp, err := SearchAcquiredSkillList(context.Background(), &BuiltinSkillIdListParams{SkillIdList: acquiredSkillIds})
		if err != nil {
			log.Errorf("Assistant服务获取Acquired Skill详情失败，assistantId: %d, error: %v", agent.Assistant.ID, err)
			return err
		}
		if acquiredSkillResp.Data != nil {
			prepareParams.AcquiredSkillList = acquiredSkillResp.Data.SkillList
		}
	}

	// 获取builtin skill详情
	if len(builtinSkillIds) > 0 {
		resp, err := SearchBuiltInSkillList(context.Background(), &BuiltinSkillIdListParams{SkillIdList: builtinSkillIds})
		if err != nil {
			log.Errorf("Assistant服务获取BuiltIn Skill详情失败，assistantId: %d, error: %v", agent.Assistant.ID, err)
			return err
		}
		prepareParams.builtinSkillList = resp.Data.SkillList
	}
	return nil
}

func (k *SkillProcess) Build(assistant *AgentInfo, prepareParams *AgentPrepareParams, agentChatParams *assistant_service.AgentDetail) error {
	var skillInfos []*assistant_service.SkillInfo
	if len(prepareParams.CustomSkillList) > 0 {
		for _, detail := range prepareParams.CustomSkillList {
			skillInfos = append(skillInfos, &assistant_service.SkillInfo{
				SkillId:    detail.SkillId,
				SkillType:  constant.SkillTypeCustom,
				Name:       detail.Name,
				Desc:       detail.Desc,
				Avatar:     detail.Avatar.Key,
				ObjectPath: detail.ObjectPath,
			})
		}
	}
	if len(prepareParams.AcquiredSkillList) > 0 {
		for _, detail := range prepareParams.AcquiredSkillList {
			skillInfos = append(skillInfos, &assistant_service.SkillInfo{
				SkillId:    detail.SkillId,
				SkillType:  constant.SkillTypeAcquired,
				Name:       detail.Name,
				Desc:       detail.Desc,
				Avatar:     detail.Avatar.Key,
				ObjectPath: detail.ObjectPath,
			})
		}
	}
	if len(prepareParams.builtinSkillList) > 0 {
		for _, skill := range prepareParams.builtinSkillList {
			skillInfos = append(skillInfos, buildBuiltInSkillDetail(skill))
		}
	}
	if agentChatParams.SkillParams == nil {
		agentChatParams.SkillParams = &assistant_service.SkillParams{}
	}
	agentChatParams.SkillParams.SkillList = skillInfos
	return nil
}

// SearchCustomSkillList 批量搜索自定义skill详情
func SearchCustomSkillList(ctx context.Context, params *BuiltinSkillIdListParams) (*CustomSkillDetailListResp, error) {
	skillConfig := config.Cfg().Skill
	if skillConfig.CustomSkillListUri == "" {
		return nil, errors.New("custom skill list uri is empty")
	}
	url, _ := net_url.JoinPath(skillConfig.Endpoint, skillConfig.CustomSkillListUri)
	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	result, err := http_client.Default().PostJson(ctx, &http_client.HttpRequestParams{
		Url:        url,
		Body:       reqBody,
		Timeout:    time.Minute,
		MonitorKey: "custom_skill_detail_list",
		LogLevel:   http_client.LogAll,
	})
	if err != nil {
		return nil, err
	}
	var detailResp CustomSkillDetailListResp
	if err = json.Unmarshal(result, &detailResp); err != nil {
		return nil, err
	}
	if detailResp.Code != 0 {
		return nil, errors.New(detailResp.Msg)
	}
	return &detailResp, nil
}

// SearchAcquiredSkillList 批量搜索我添加skill详情
func SearchAcquiredSkillList(ctx context.Context, params *BuiltinSkillIdListParams) (*AcquiredSkillDetailListResp, error) {
	skillConfig := config.Cfg().Skill
	if skillConfig.AcquiredSkillListUri == "" {
		return nil, errors.New("acquired skill list uri is empty")
	}
	url, _ := net_url.JoinPath(skillConfig.Endpoint, skillConfig.AcquiredSkillListUri)
	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	result, err := http_client.Default().PostJson(ctx, &http_client.HttpRequestParams{
		Url:        url,
		Body:       reqBody,
		Timeout:    time.Minute,
		MonitorKey: "acquired_skill_detail_list",
		LogLevel:   http_client.LogAll,
	})
	if err != nil {
		return nil, err
	}
	var detailResp AcquiredSkillDetailListResp
	if err = json.Unmarshal(result, &detailResp); err != nil {
		return nil, err
	}
	if detailResp.Code != 0 {
		return nil, errors.New(detailResp.Msg)
	}
	return &detailResp, nil
}

// SearchBuiltInSkillList 批量搜索内置skill详情
func SearchBuiltInSkillList(ctx context.Context, params *BuiltinSkillIdListParams) (*SkillDetailListResp, error) {
	skillConfig := config.Cfg().Skill
	url, _ := net_url.JoinPath(skillConfig.Endpoint, skillConfig.BuiltInSkillListUri)
	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	result, err := http_client.Default().PostJson(ctx, &http_client.HttpRequestParams{
		Url:        url,
		Body:       reqBody,
		Timeout:    time.Minute,
		MonitorKey: "builtin_skill_detail_list",
		LogLevel:   http_client.LogAll,
	})
	if err != nil {
		return nil, err
	}
	var detailResp SkillDetailListResp
	if err = json.Unmarshal(result, &detailResp); err != nil {
		return nil, err
	}
	if detailResp.Code != 0 {
		return nil, errors.New(detailResp.Msg)
	}
	return &detailResp, nil
}

// buildBuiltInSkillDetail 构建内置skill详情
func buildBuiltInSkillDetail(skill *SkillDetail) *assistant_service.SkillInfo {
	return &assistant_service.SkillInfo{
		SkillId:    skill.SkillId,
		SkillType:  constant.SkillTypeBuiltIn,
		Name:       skill.Name,
		Desc:       skill.Desc,
		Avatar:     skill.Avatar.Key,
		ObjectPath: skill.SkillPath,
	}
}

func buildAssistantSkills(agent *AgentInfo, clientInfo *ClientInfo) []*model.AssistantSkill {
	if agent.Draft {
		list, status := clientInfo.Cli.GetAssistantSkillList(context.Background(), agent.Assistant.ID)
		if status != nil {
			log.Errorf("GetAssistantSkillList error: %v", status)
			return nil
		}
		return list
	}
	var skillList []*model.AssistantSkill
	if agent.AssistantSnapshot.AssistantSkillConfig != "" {
		if err := json.Unmarshal([]byte(agent.AssistantSnapshot.AssistantSkillConfig), &skillList); err != nil {
			log.Errorf("GetAssistantSnapshotSkillList error: %v", err)
			return nil
		}
	}
	return skillList
}

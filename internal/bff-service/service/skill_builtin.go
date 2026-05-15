package service

import (
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/gin-gonic/gin"
	"strings"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
)

func GetAgentSkillDetail(ctx *gin.Context, skillId string) (*response.SkillDetail, error) {
	skillsCfg, exist := config.Cfg().AgentSkill(skillId)
	if !exist {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "bff_agent_skill_detail", "get skill detail empty")
	}
	return buildSkillTempDetail(skillsCfg, true), nil
}

func GetAgentSkillListDetail(ctx *gin.Context, skillIdList []string) (*response.SkillDetailListResp, error) {
	var skillDetailList []*response.SkillDetail
	for _, skillId := range skillIdList {
		skillsCfg, exist := config.Cfg().AgentSkill(skillId)
		if !exist {
			continue
		}
		detail := buildSkillTempDetail(skillsCfg, false)
		detail.SkillPath = skillsCfg.SkillPath
		skillDetailList = append(skillDetailList, detail)
	}
	return &response.SkillDetailListResp{SkillList: skillDetailList}, nil
}

func GetBuiltinSkillList(ctx *gin.Context, name string) (*response.ListResult, error) {
	var list []*response.SkillDetail
	for _, skillsCfg := range config.Cfg().AgentSkills {
		if name != "" && !strings.Contains(skillsCfg.Name, name) {
			continue
		}
		list = append(list, buildSkillTempDetail(*skillsCfg, false))
	}
	return &response.ListResult{
		List:  list,
		Total: int64(len(list)),
	}, nil
}

func GetBuiltinSkillDetail(ctx *gin.Context, userId, orgId, skillId string) (*response.SkillDetail, error) {
	skillsCfg, exist := config.Cfg().AgentSkill(skillId)
	if !exist {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "skill not found in builtin skills")
	}

	detail := buildSkillTempDetail(skillsCfg, true)
	configResp, err := mcp.GetBuiltinSkillVars(ctx.Request.Context(), &mcp_service.GetBuiltinSkillVarsReq{
		SkillId:  skillId,
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		return nil, err
	}
	if configResp != nil {
		detail.Variables = append(detail.Variables, toSkillVariables(configResp.Variables)...)
	}
	return detail, nil
}

func DownloadBuiltinSkill(ctx *gin.Context, skillId string) ([]byte, error) {
	skillsCfg, exist := config.Cfg().AgentSkill(skillId)
	if !exist {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "skill not found in builtin skills")
	}
	return skillsCfg.AgentSkillZipToBytes(skillId)
}

// --- internal ---

func buildSkillTempDetail(skillsCfg config.SkillsConfig, needMd bool) *response.SkillDetail {
	iconUrl := config.Cfg().DefaultIcon.SkillIcon
	if skillsCfg.Avatar != "" {
		iconUrl = skillsCfg.Avatar
	}
	ret := &response.SkillDetail{
		SkillId: skillsCfg.SkillId,
		Author:  skillsCfg.Author,
		Avatar:  request.Avatar{Path: iconUrl},
		Name:    skillsCfg.Name,
		Desc:    skillsCfg.Desc,
	}
	if needMd {
		ret.SkillMarkdown = string(skillsCfg.SkillMarkdown)
	}
	return ret
}

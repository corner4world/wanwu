package service

import (
	"bytes"
	"path/filepath"
	"strings"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/pkg/constant"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/minio"
	"github.com/gin-gonic/gin"
)

// GetSquareSkillList 探索广场-skill列表（内置skill配置 + isShared计算）
func GetSquareSkillList(ctx *gin.Context, userId, orgId, name string) (*response.ListResult, error) {
	var skillsCfgList []*config.SkillsConfig
	for _, skillsCfg := range config.Cfg().AgentSkills {
		if name != "" && !strings.Contains(skillsCfg.Name, name) {
			continue
		}
		skillsCfgList = append(skillsCfgList, skillsCfg)
	}

	// 查询当前用户已添加的 acquired skill，计算 isShared
	sharedMap := make(map[string]bool)
	if len(skillsCfgList) > 0 {
		acquiredResp, err := mcp.AcquiredSkillGetList(ctx.Request.Context(), &mcp_service.AcquiredSkillGetListReq{
			Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
		})
		if err != nil {
			return nil, err
		}
		if acquiredResp != nil {
			for _, skill := range acquiredResp.List {
				sharedMap[skill.SquareSkillId] = true
			}
		}
	}

	list := make([]*response.SquareSkillInfo, 0, len(skillsCfgList))
	for _, skillsCfg := range skillsCfgList {
		iconUrl := config.Cfg().DefaultIcon.SkillIcon
		if skillsCfg.Avatar != "" {
			iconUrl = skillsCfg.Avatar
		}
		list = append(list, &response.SquareSkillInfo{
			SkillId:  skillsCfg.SkillId,
			Name:     skillsCfg.Name,
			Avatar:   request.Avatar{Path: iconUrl},
			Author:   skillsCfg.Author,
			Desc:     skillsCfg.Desc,
			IsShared: sharedMap[skillsCfg.SkillId],
		})
	}

	return &response.ListResult{
		List:  list,
		Total: int64(len(list)),
	}, nil
}

// ShareSquareSkill 探索广场-添加skill到资源库
func ShareSquareSkill(ctx *gin.Context, userId, orgId, skillId string) error {
	skillsCfg, exist := config.Cfg().AgentSkill(skillId)
	if !exist {
		return grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "skill not found in builtin skills")
	}

	// 将内置skill打包上传到MinIO永久路径
	var objectPath string
	zipData, err := skillsCfg.AgentSkillZipToBytes(skillId)
	if err != nil {
		return err
	}
	if len(zipData) > 0 {
		fileName, _, uploadErr := minio.UploadFileCommon(ctx.Request.Context(), bytes.NewReader(zipData), customSkillFileType, -1, true)
		if uploadErr == nil {
			objectPath = filepath.Join(minio.BucketFileUpload, minio.DirFileNotExpire, fileName)
		}
	}

	_, err = mcp.AcquiredSkillCreate(ctx.Request.Context(), &mcp_service.AcquiredSkillCreateReq{
		Identity:      &mcp_service.Identity{UserId: userId, OrgId: orgId},
		Name:          skillsCfg.Name,
		Avatar:        skillsCfg.Avatar,
		SquareSkillId: skillId,
		Author:        skillsCfg.Author,
		Desc:          skillsCfg.Desc,
		ObjectPath:    objectPath,
		AcquiredType:  constant.SkillTypeBuiltIn,
		Markdown:      string(skillsCfg.SkillMarkdown),
	})
	return err
}

// GetSquareSkillDetail 探索广场-skill详情
func GetSquareSkillDetail(ctx *gin.Context, userId, orgId, skillId string) (*response.SquareSkillDetail, error) {
	skillsCfg, exist := config.Cfg().AgentSkill(skillId)
	if !exist {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "skill not found in builtin skills")
	}

	// 查询当前用户是否已添加该skill
	isShared := false
	acquiredResp, err := mcp.AcquiredSkillGetList(ctx.Request.Context(), &mcp_service.AcquiredSkillGetListReq{
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		return nil, err
	}
	if acquiredResp != nil {
		for _, skill := range acquiredResp.List {
			if skill.SquareSkillId == skillId {
				isShared = true
				break
			}
		}
	}

	iconUrl := config.Cfg().DefaultIcon.SkillIcon
	if skillsCfg.Avatar != "" {
		iconUrl = skillsCfg.Avatar
	}
	return &response.SquareSkillDetail{
		SquareSkillInfo: response.SquareSkillInfo{
			SkillId:  skillsCfg.SkillId,
			Name:     skillsCfg.Name,
			Avatar:   request.Avatar{Path: iconUrl},
			Author:   skillsCfg.Author,
			Desc:     skillsCfg.Desc,
			IsShared: isShared,
		},
		SkillMarkdown: string(skillsCfg.SkillMarkdown),
	}, nil
}

// DownloadSquareSkill 探索广场-下载skill
func DownloadSquareSkill(ctx *gin.Context, skillId string) ([]byte, error) {
	skillsCfg, exist := config.Cfg().AgentSkill(skillId)
	if !exist {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "skill not found in builtin skills")
	}
	return skillsCfg.AgentSkillZipToBytes(skillId)
}

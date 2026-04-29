package service

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	minio_util "github.com/UnicomAI/wanwu/internal/bff-service/pkg/minio-util"
	"github.com/UnicomAI/wanwu/pkg/constant"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	wga_option "github.com/UnicomAI/wanwu/pkg/wga/wga-option"
	"github.com/gin-gonic/gin"
)

// --- internal wga skill ---

// checkWgaSkillConfig 校验wga Skill配置（用于更新配置）
func checkWgaSkillConfig(ctx *gin.Context, userId, orgId string, skillList []*assistant_service.WgaConfigSkill) error {
	if len(skillList) == 0 {
		return nil
	}

	var customSkillIds []string
	for _, s := range skillList {
		switch s.SkillType {
		case constant.SkillTypeCustom:
			customSkillIds = append(customSkillIds, s.SkillId)
		default:
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("invalid skill type: %s", s.SkillType))
		}
	}

	validIds, err := getValidSkillIds(ctx, customSkillIds)
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "skill not found")
	}

	for _, s := range skillList {
		if !validIds[s.SkillId] {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("skill not found: %s", s.SkillId))
		}
	}
	return nil
}

func buildWgaSkillOptions(ctx *gin.Context, userId, orgId, threadId, runId string, skillList []*assistant_service.WgaConfigSkill) ([]wga_option.Option, error) {
	if len(skillList) == 0 {
		return nil, nil
	}

	var customSkillIds []string
	for _, s := range skillList {
		switch s.SkillType {
		case constant.SkillTypeCustom:
			customSkillIds = append(customSkillIds, s.SkillId)
		default:
			return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("invalid skill type: %s", s.SkillType))
		}
	}

	resp, err := mcp.GetCustomSkillDetailByIdList(ctx.Request.Context(), &mcp_service.CustomSkillDetailByIdListReq{
		SkillIds: customSkillIds,
	})
	if err != nil {
		return nil, err
	}

	var opts []wga_option.Option
	for _, s := range resp.SkillDetails {
		skillUrl, _ := url.JoinPath("http://", config.Cfg().Minio.Endpoint, s.ObjectPath)

		b, skillZipName, err := minio_util.DownloadFile(ctx.Request.Context(), skillUrl)
		if err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to download skill file from %s: %v", skillUrl, err))
		}
		skillTempDir := filepath.Join(os.TempDir(), "wga", threadId, runId, "skills", s.SkillId)
		if err := os.MkdirAll(skillTempDir, 0755); err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to create skill temp dir %s: %v", skillTempDir, err))
		}
		skillZipPath := filepath.Join(skillTempDir, skillZipName)
		if err := os.WriteFile(skillZipPath, b, 0644); err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to write skill zip %s: %v", skillZipPath, err))
		}
		if _, err := util.UnzipDir(ctx.Request.Context(), skillZipPath, skillTempDir); err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to unzip skill %s: %v", skillZipPath, err))
		}
		if err := util.DeleteFile(skillZipPath); err != nil {
			log.Warnf("failed to delete skill zip file %s: %v", skillZipPath, err)
		}
		opts = append(opts, wga_option.WithSkill(wga_option.Skill{Dir: skillTempDir}))
	}

	return opts, nil
}

// getValidSkillIds 批量获取有效的Skill ID映射
func getValidSkillIds(ctx *gin.Context, skillIds []string) (map[string]bool, error) {
	if len(skillIds) == 0 {
		return make(map[string]bool), nil
	}
	resp, err := mcp.GetCustomSkillDetailByIdList(ctx.Request.Context(), &mcp_service.CustomSkillDetailByIdListReq{
		SkillIds: skillIds,
	})
	if err != nil {
		return nil, err
	}
	validIds := make(map[string]bool)
	for _, s := range resp.SkillDetails {
		validIds[s.SkillId] = true
	}
	return validIds, nil
}

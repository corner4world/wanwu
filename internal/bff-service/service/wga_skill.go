package service

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
		var skillTempDir string

		if s.ObjectPath != "" {
			// 已发布的 skill：从 minio 下载
			skillUrl, _ := url.JoinPath("http://", config.Cfg().Minio.Endpoint, s.ObjectPath)
			b, skillZipName, err := minio_util.DownloadFile(ctx.Request.Context(), skillUrl)
			if err != nil {
				return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to download skill file from %s: %v", skillUrl, err))
			}
			skillTempDir = filepath.Join(os.TempDir(), "wga", threadId, runId, "skills", s.SkillId)
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
		} else {
			// 未发布的 skill（草稿）：从 workspace 目录获取
			skillDir, err := findFirstCustomSkillDir(s.SkillId)
			if err != nil {
				return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to find skill workspace for %s: %v", s.SkillId, err))
			}
			if err := ensureNoSymlink(skillDir); err != nil {
				return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("custom skill %s contains symlink: %v", s.SkillId, err))
			}
			// 直接使用 workspace 目录
			skillTempDir = skillDir
		}

		opts = append(opts, wga_option.WithSkill(wga_option.Skill{Dir: skillTempDir, Variables: getWgaSkillVariablesByCustomSkill(s)}))
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

func getWgaSkillVariablesByCustomSkill(customSkill *mcp_service.CustomSkill) []wga_option.SkillVariable {
	var variables []wga_option.SkillVariable
	for _, v := range customSkill.Variables {
		variables = append(variables, wga_option.SkillVariable{
			Name:          v.Name,
			Description:   v.Desc,
			VariableKey:   v.VariableKey,
			VariableValue: v.VariableValue,
		})
	}
	return variables
}

// buildWgaSkillVariablesMessage 构建技能变量的系统消息，用于 Skill Preview Agent 模式
func buildWgaSkillVariablesMessage(customSkill *mcp_service.CustomSkill) string {
	var buf strings.Builder

	// 技能基本信息
	buf.WriteString("# 当前工作空间技能信息\n\n")
	if customSkill.Name != "" {
		buf.WriteString(fmt.Sprintf("**技能名称**: %s\n", customSkill.Name))
	}
	if customSkill.Desc != "" {
		buf.WriteString(fmt.Sprintf("**技能描述**: %s\n", customSkill.Desc))
	}
	buf.WriteString("\n")

	// 用户定义的变量
	if len(customSkill.Variables) > 0 {
		buf.WriteString("## 用户定义的变量\n\n")
		buf.WriteString("以下变量已为当前技能配置：\n\n")
		for _, v := range customSkill.Variables {
			// 转义反引号
			escapedKey := strings.ReplaceAll(v.VariableKey, "`", "\\`")
			escapedValue := strings.ReplaceAll(v.VariableValue, "`", "\\`")

			if v.Desc != "" {
				escapedDesc := strings.ReplaceAll(v.Desc, "`", "\\`")
				fmt.Fprintf(&buf, "- **%s** (%s): `%s` = `%s`\n",
					v.Name, escapedDesc, escapedKey, escapedValue)
			} else {
				fmt.Fprintf(&buf, "- **%s**: `%s` = `%s`\n",
					v.Name, escapedKey, escapedValue)
			}
		}
	}

	return buf.String()
}

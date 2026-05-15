package service

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	minio_util "github.com/UnicomAI/wanwu/internal/bff-service/pkg/minio-util"
	"github.com/UnicomAI/wanwu/pkg/constant"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/minio"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
)

const (
	customSkillFileType = ".zip"
)

func CreateCustomSkill(ctx *gin.Context, userId, orgId string, avatarKey, author, zipUrl, saveId, sourceType string) (*response.CustomSkillIDResp, error) {
	var skillName, skillDesc string

	if zipUrl != "" {
		// 下载文件
		data, err := minio_util.DownloadFileDirect(ctx.Request.Context(), zipUrl)
		if err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("download skill zip err: %v", err))
		}

		// 解压并查找SKILL.md文件，提取name和description
		_, fm, err := util.ExtractSkillMarkdownFromZip(data)
		if err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFSkillParse, err.Error())
		}

		// 如果从markdown中提取到了name和desc，使用这些值
		skillName = fm.Name
		skillDesc = fm.Description

		_, _, err = minio.UploadFileCommon(ctx.Request.Context(), bytes.NewReader(data), customSkillFileType, -1, true)
		if err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
		}
	}

	createResp, err := mcp.CustomSkillCreate(ctx.Request.Context(), &mcp_service.CustomSkillCreateReq{
		Name:       skillName,
		Avatar:     avatarKey,
		Author:     author,
		Desc:       skillDesc,
		SourceType: sourceType,
		Identity:   &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		return nil, err
	}

	return &response.CustomSkillIDResp{SkillId: createResp.SkillId}, nil
}

func GetCustomSkill(ctx *gin.Context, userId, orgId, skillId string) (*response.CustomSkillDetail, error) {
	resp, err := mcp.CustomSkillGet(ctx.Request.Context(), &mcp_service.CustomSkillGetReq{
		SkillId: skillId,
	})
	if err != nil {
		return nil, err
	}

	return &response.CustomSkillDetail{
		SkillId:   resp.SkillId,
		Name:      resp.Name,
		Avatar:    cacheSkillAvatar(ctx, resp.Avatar),
		Author:    resp.Author,
		Desc:      resp.Desc,
		Variables: toSkillVariables(resp.Variables),
		ThreadID:  resp.WgaThreadId,
		PreviewID: resp.PreviewThreadId,
	}, nil
}

func ensureLegacyCustomSkillWorkspace(ctx *gin.Context, skill *mcp_service.CustomSkill) error {
	if skill == nil {
		return nil
	}
	zipURL := strings.TrimSpace(skill.ObjectPath)
	if zipURL == "" {
		return nil
	}

	workspaceExists, workspaceDir, err := customSkillOverwriteWorkspaceExists(skill.SkillId)
	if err != nil {
		log.Errorf("[wga-skill-legacy] skill %v check overwrite workspace err: %v", skill.SkillId, err)
		return err
	}
	if workspaceExists {
		log.Infof("[wga-skill-legacy] skill %v overwrite workspace exists, workspaceDir=%s, skip import", skill.SkillId, workspaceDir)
		return nil
	}

	log.Infof("[wga-skill-legacy] skill %v overwrite workspace not found, expectedWorkspaceDir=%s, start import from objectPath", skill.SkillId, workspaceDir)
	if err := importLegacyCustomSkillWorkspace(ctx, skill.SkillId, zipURL); err != nil {
		log.Errorf("[wga-skill-legacy] skill %v import overwrite workspace err: %v", skill.SkillId, err)
		return err
	}
	workspaceExists, workspaceDir, err = customSkillOverwriteWorkspaceExists(skill.SkillId)
	if err != nil {
		log.Errorf("[wga-skill-legacy] skill %v recheck overwrite workspace err: %v", skill.SkillId, err)
		return err
	}
	if !workspaceExists {
		log.Errorf("[wga-skill-legacy] skill %v import finished but overwrite workspace still not found, expectedWorkspaceDir=%s", skill.SkillId, workspaceDir)
		return grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("custom skill workspace import finished but workspace not found: %s", skill.SkillId))
	}
	log.Infof("[wga-skill-legacy] skill %v import overwrite workspace success, workspaceDir=%s", skill.SkillId, workspaceDir)
	return nil
}

func customSkillOverwriteWorkspaceExists(skillId string) (bool, string, error) {
	store, err := NewGeneralAgentSkillWorkspaceStore(skillId)
	if err != nil {
		return false, "", err
	}
	ok, info, err := store.GetLastRunDir()
	if err != nil {
		return false, info.Dir, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if !ok {
		return false, info.Dir, nil
	}
	stat, err := os.Stat(info.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, info.Dir, nil
		}
		return false, info.Dir, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("stat custom skill workspace %s err: %v", info.Dir, err))
	}
	if !stat.IsDir() {
		return false, info.Dir, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("custom skill workspace is not a directory: %s", info.Dir))
	}
	return true, info.Dir, nil
}

func importLegacyCustomSkillWorkspace(ctx *gin.Context, skillId, zipURL string) error {
	store, err := NewGeneralAgentSkillWorkspaceStore(skillId)
	if err != nil {
		return err
	}
	dirs, err := PrepareWgaWorkspaceDirs(store, util.GenUUID(), true)
	if err != nil {
		return err
	}
	log.Infof("[wga-skill-legacy] skill %v prepared overwrite workspace, outputDir=%s", skillId, dirs.OutputDir)
	skillDir := filepath.Join(dirs.OutputDir, generalAgentSkillImportDirName)
	data, err := downloadCustomSkillZip(ctx, zipURL)
	if err != nil {
		_ = CleanupWgaWorkspace(store)
		return grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("download skill zip err: %v", err))
	}
	log.Infof("[wga-skill-legacy] skill %v downloaded legacy skill zip, size=%d, skillDir=%s", skillId, len(data), skillDir)
	if _, err := importSkillDataIntoWorkspace(data, skillDir); err != nil {
		_ = CleanupWgaWorkspace(store)
		return err
	}
	return nil
}

func downloadCustomSkillZip(ctx *gin.Context, zipURL string) ([]byte, error) {
	zipURL = strings.TrimSpace(zipURL)
	if isHTTPURL(zipURL) {
		if data, err := minio_util.DownloadFileDirect(ctx.Request.Context(), zipURL); err == nil {
			return data, nil
		}
		data, _, err := minio_util.DownloadFile(ctx.Request.Context(), zipURL)
		return data, err
	}
	data, _, err := minio_util.DownloadFile(ctx.Request.Context(), buildMinioObjectURL(zipURL))
	return data, err
}

func buildMinioObjectURL(objectPath string) string {
	endpoint := strings.TrimSpace(config.Cfg().Minio.Endpoint)
	if endpoint == "" {
		return objectPath
	}
	if isHTTPURL(endpoint) {
		return strings.TrimRight(endpoint, "/") + "/" + strings.TrimLeft(objectPath, "/")
	}
	return "http://" + strings.TrimRight(endpoint, "/") + "/" + strings.TrimLeft(objectPath, "/")
}

func isHTTPURL(rawURL string) bool {
	rawURL = strings.ToLower(strings.TrimSpace(rawURL))
	return strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")
}

func DeleteCustomSkill(ctx *gin.Context, userId, orgId, skillId string) error {
	skill, err := mcp.CustomSkillGet(ctx.Request.Context(), &mcp_service.CustomSkillGetReq{
		SkillId: skillId,
	})
	if err != nil {
		return err
	}

	if skill.WgaThreadId != "" {
		if err := deleteGeneralAgentConversationForSkillDelete(ctx, userId, orgId, request.DeleteGeneralAgentConversationReq{ThreadID: skill.WgaThreadId}); err != nil {
			return err
		}
	}
	if skill.PreviewThreadId != "" {
		if err := deleteWgaConversationHistory(ctx, userId, orgId, skill.PreviewThreadId); err != nil {
			return err
		}
	}
	if err := cleanupCustomSkillWorkspace(skill.SkillId); err != nil {
		return err
	}

	_, err = mcp.CustomSkillDelete(ctx.Request.Context(), &mcp_service.CustomSkillDeleteReq{
		SkillId: skillId,
	})
	return err
}

func GetCustomSkillList(ctx *gin.Context, userId, orgId, name string) (*response.ListResult, error) {
	resp, err := mcp.CustomSkillGetList(ctx.Request.Context(), &mcp_service.CustomSkillGetListReq{
		Name:     name,
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		return nil, err
	}

	customSkillList := make([]*response.CustomSkillDetail, 0, len(resp.List))
	for _, skill := range resp.List {
		customSkillList = append(customSkillList, toCustomSkill(ctx, skill))
	}

	return &response.ListResult{
		List:  customSkillList,
		Total: resp.Total,
	}, nil
}

func GetCustomSkillListDetail(ctx *gin.Context, skillIdList []string) (*response.CustomSkillDetailListResp, error) {
	skillIdList = uniqueSkillIDs(skillIdList)
	resp, err := mcp.GetCustomSkillDetailByIdList(ctx.Request.Context(), &mcp_service.CustomSkillDetailByIdListReq{
		SkillIds: skillIdList,
	})
	if err != nil {
		return nil, err
	}

	skillMap := make(map[string]*mcp_service.CustomSkill)
	if resp != nil {
		for _, skill := range resp.SkillDetails {
			if skill != nil {
				skillMap[skill.SkillId] = skill
			}
		}
	}

	skillDetailList := make([]*response.CustomSkillListDetail, 0, len(skillIdList))
	for _, skillId := range skillIdList {
		skill := skillMap[skillId]
		if skill == nil {
			continue
		}
		objectPath, err := buildCustomSkillListDetailObjectPath(ctx, skill)
		if err != nil {
			return nil, err
		}
		skillDetailList = append(skillDetailList, &response.CustomSkillListDetail{
			SkillId:    skill.SkillId,
			Name:       skill.Name,
			Avatar:     cacheSkillAvatar(ctx, skill.Avatar),
			Author:     skill.Author,
			Desc:       skill.Desc,
			ObjectPath: objectPath,
			// Variables:  toSkillVariables(skill.Variables),
		})
	}

	return &response.CustomSkillDetailListResp{SkillList: skillDetailList}, nil
}

func buildCustomSkillListDetailObjectPath(ctx *gin.Context, skill *mcp_service.CustomSkill) (string, error) {
	if exists, _, err := customSkillOverwriteWorkspaceExists(skill.SkillId); err != nil {
		return "", err
	} else if !exists {
		return skill.ObjectPath, nil
	}
	return buildCustomSkillExpireObjectPath(ctx, skill.SkillId)
}

func uniqueSkillIDs(skillIdList []string) []string {
	seen := make(map[string]struct{}, len(skillIdList))
	result := make([]string, 0, len(skillIdList))
	for _, skillId := range skillIdList {
		if _, ok := seen[skillId]; ok {
			continue
		}
		seen[skillId] = struct{}{}
		result = append(result, skillId)
	}
	return result
}

func buildCustomSkillExpireObjectPath(ctx *gin.Context, skillId string) (string, error) {
	skillDir, err := findFirstCustomSkillDir(skillId)
	if err != nil {
		return "", err
	}
	if err := ensureNoSymlink(skillDir); err != nil {
		return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("custom skill %s contains symlink: %v", skillId, err))
	}
	zipBytes, err := util.ZipDir(skillDir + string(os.PathSeparator) + ".")
	if err != nil {
		return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("zip custom skill %s err: %v", skillId, err))
	}
	fileName, _, err := minio.UploadFileCommon(ctx.Request.Context(), bytes.NewReader(zipBytes), customSkillFileType, int64(len(zipBytes)), false)
	if err != nil {
		return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("upload custom skill %s zip err: %v", skillId, err))
	}
	return path.Join(minio.BucketFileUpload, minio.DirFileExpire, fileName), nil
}

func findFirstCustomSkillDir(skillId string) (string, error) {
	store, err := NewGeneralAgentSkillWorkspaceStore(skillId)
	if err != nil {
		return "", err
	}
	workspaceDir := GetWgaWorkspaceThreadDir(store)
	if workspaceDir == "" {
		return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("custom skill workspace not found: %s", skillId))
	}
	stat, err := os.Stat(workspaceDir)
	if err != nil {
		return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("custom skill workspace not found: %s", skillId))
	}
	if !stat.IsDir() {
		return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("custom skill workspace is not a directory: %s", workspaceDir))
	}
	return findFirstCustomSkillDirInWorkspace(workspaceDir, skillId)
}

func findFirstCustomSkillDirInWorkspace(workspaceDir, skillId string) (string, error) {
	var skillDir string
	rootSkillFound := false
	err := filepath.WalkDir(workspaceDir, func(currentPath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if currentPath == workspaceDir {
				return nil
			}
			if shouldSkipCustomSkillWorkspaceDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "SKILL.md" {
			return nil
		}
		parentDir := filepath.Dir(currentPath)
		if parentDir == workspaceDir {
			rootSkillFound = true
			return filepath.SkipAll
		}
		skillDir = parentDir
		return filepath.SkipAll
	})
	if err != nil {
		return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to scan custom skill workspace %s: %v", workspaceDir, err))
	}
	if skillDir != "" {
		return skillDir, nil
	}
	if rootSkillFound {
		return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("custom skill SKILL.md is at workspace root, refuse to zip workspace: %s", skillId))
	}
	return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("custom skill SKILL.md not found in workspace: %s", skillId))
}

func shouldSkipCustomSkillWorkspaceDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "tmp", "input", "output":
		return true
	default:
		return false
	}
}

func ensureNoSymlink(dir string) error {
	return filepath.WalkDir(dir, func(currentPath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not supported: %s", currentPath)
		}
		return nil
	})
}

func toCustomSkill(ctx *gin.Context, skill *mcp_service.CustomSkill) *response.CustomSkillDetail {
	if skill == nil {
		return nil
	}

	return &response.CustomSkillDetail{
		SkillId:   skill.SkillId,
		Name:      skill.Name,
		Avatar:    cacheSkillAvatar(ctx, skill.Avatar),
		Author:    skill.Author,
		Desc:      skill.Desc,
		ThreadID:  skill.WgaThreadId,
		PreviewID: skill.PreviewThreadId,
	}
}

func CheckCustomSkill(ctx *gin.Context, userId, orgId, zipUrl string) (*response.CustomSkillCheckResp, error) {
	// 下载文件
	data, err := minio_util.DownloadFileDirect(ctx.Request.Context(), zipUrl)
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("download skill zip err: %v", err))
	}

	// 解压并查找SKILL.md文件，验证zip包是否有效
	_, fm, err := util.ExtractSkillMarkdownFromZip(data)
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFSkillParse, err.Error())
	}

	return &response.CustomSkillCheckResp{
		Name: fm.Name,
		Desc: fm.Description,
	}, nil
}

func GetSkillSelect(ctx *gin.Context, userId, orgId, name, skillType string) (*response.ListResult, error) {
	var allSkills []*response.SkillInfo

	// 内建 skills
	if skillType == "" || skillType == constant.SkillTypeBuiltIn {
		for _, skillsCfg := range config.Cfg().AgentSkills {
			if name != "" && !strings.Contains(skillsCfg.Name, name) {
				continue
			}
			iconUrl := config.Cfg().DefaultIcon.SkillIcon
			if skillsCfg.Avatar != "" {
				iconUrl = skillsCfg.Avatar
			}
			allSkills = append(allSkills, &response.SkillInfo{
				SkillId:   skillsCfg.SkillId,
				SkillName: skillsCfg.Name,
				SkillType: constant.SkillTypeBuiltIn,
				Desc:      skillsCfg.Desc,
				Author:    skillsCfg.Author,
				Avatar:    request.Avatar{Path: iconUrl},
			})
		}
	}

	// 自定义 skills
	if skillType == "" || skillType == constant.SkillTypeCustom {
		customResp, err := mcp.CustomSkillGetList(ctx.Request.Context(), &mcp_service.CustomSkillGetListReq{
			Name:     name,
			Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
		})
		if err != nil {
			return nil, err
		}

		for _, skill := range customResp.List {
			if !validCustomSkillForSelect(skill) {
				continue
			}
			allSkills = append(allSkills, &response.SkillInfo{
				SkillId:   skill.SkillId,
				SkillName: skill.Name,
				SkillType: constant.SkillTypeCustom,
				Desc:      skill.Desc,
				Author:    skill.Author,
				Avatar:    cacheSkillAvatar(ctx, skill.Avatar),
			})
		}
	}

	return &response.ListResult{
		List:  allSkills,
		Total: int64(len(allSkills)),
	}, nil
}

func validCustomSkillForSelect(skill *mcp_service.CustomSkill) bool {
	if skill == nil {
		return false
	}
	return strings.TrimSpace(skill.Name) != "" && strings.TrimSpace(skill.Desc) != ""
}

func deleteWgaConversationHistory(ctx *gin.Context, userId, orgId, threadId string) error {
	threadId = strings.TrimSpace(threadId)
	if threadId == "" {
		return nil
	}
	_, err := assistant.DeleteFromES(ctx.Request.Context(), &assistant_service.DeleteFromESReq{
		IndexName: wgaConversationHistoryEventESIndexName,
		Conditions: map[string]string{
			"threadId": threadId,
			"userId":   userId,
			"orgId":    orgId,
		},
	})
	if err != nil && !wgaConversationHistoryEventESIndexNotFound(err) {
		return err
	}
	return nil
}

func cleanupCustomSkillWorkspace(skillId string) error {
	store, err := NewGeneralAgentSkillWorkspaceStore(skillId)
	if err != nil {
		log.Warnf("[wga-skill] skill %v create persistent store for cleanup err: %v", skillId, err)
		return nil
	}
	return CleanupWgaWorkspace(store)
}

package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	"github.com/UnicomAI/wanwu/api/proto/common"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	minio_util "github.com/UnicomAI/wanwu/internal/bff-service/pkg/minio-util"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
)

const generalAgentSkillImportDirName = "skill"

type importedSkillRoot struct {
	zipPath string
	name    string
}

func ImportGeneralAgentSkillConversation(ctx *gin.Context, userId, orgId string, req request.ImportGeneralAgentSkillConversationReq) (*response.ImportGeneralAgentSkillConversationResp, error) {
	if err := checkModelConfig(ctx, req.ModelConfig); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ZipUrl) == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "zipUrl is required")
	}
	modelConfigString, err := req.ModelConfig.ConfigString()
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, err.Error())
	}

	previewID := util.GenUUID()

	threadResp, err := assistant.WgaConversationCreate(ctx.Request.Context(), &assistant_service.WgaConversationCreateReq{
		Prompt: "Import Skill",
		ModelConfig: &common.AppModelConfig{
			ModelId:   req.ModelConfig.ModelId,
			Provider:  req.ModelConfig.Provider,
			Model:     req.ModelConfig.Model,
			ModelType: req.ModelConfig.ModelType,
			Config:    modelConfigString,
		},
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}

	author := req.Author
	if author == "" {
		author = skillConversationAuthor
	}
	customSkillResp, err := mcp.CustomSkillCreate(ctx.Request.Context(), &mcp_service.CustomSkillCreateReq{
		Name:            "Import Skill",
		Avatar:          req.Avatar.Key,
		Author:          author,
		WgaThreadId:     threadResp.ThreadId,
		PreviewThreadId: previewID,
		SourceType:      customSkillSourceTypeImport,
		Identity:        &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		_, _ = assistant.WgaConversationDelete(ctx.Request.Context(), &assistant_service.WgaConversationDeleteReq{
			ThreadId: threadResp.ThreadId,
			Identity: &assistant_service.Identity{
				UserId: userId,
				OrgId:  orgId,
			},
		})
		return nil, err
	}
	customSkillID := customSkillResp.SkillId

	store, err := NewGeneralAgentSkillWorkspaceStore(customSkillID)
	if err != nil {
		rollbackImportedSkillConversation(ctx, userId, orgId, threadResp.ThreadId, customSkillID)
		return nil, err
	}
	skillDir := filepath.Join(GetWgaWorkspaceThreadDir(store), generalAgentSkillImportDirName)
	fm, err := importSkillIntoWorkspace(ctx, req.ZipUrl, skillDir)
	if err != nil {
		rollbackImportedSkillConversation(ctx, userId, orgId, threadResp.ThreadId, customSkillID)
		return nil, err
	}
	_, err = mcp.UpdateCustomSkillBasicMeta(ctx.Request.Context(), &mcp_service.UpdateCustomSkillBasicMetaReq{
		SkillId: customSkillID,
		Name:    fm.Name,
		Desc:    fm.Description,
	})
	if err != nil {
		rollbackImportedSkillConversation(ctx, userId, orgId, threadResp.ThreadId, customSkillID)
		return nil, err
	}

	return &response.ImportGeneralAgentSkillConversationResp{
		CustomSkillID: customSkillID,
		ThreadID:      threadResp.ThreadId,
		PreviewID:     previewID,
	}, nil
}

func rollbackImportedSkillConversation(ctx *gin.Context, userId, orgId, threadId, customSkillId string) {
	if customSkillId != "" {
		_, _ = mcp.CustomSkillDelete(ctx.Request.Context(), &mcp_service.CustomSkillDeleteReq{SkillId: customSkillId})
		_ = cleanupCustomSkillWorkspace(customSkillId)
	}
	if threadId != "" {
		_, _ = assistant.WgaConversationDelete(ctx.Request.Context(), &assistant_service.WgaConversationDeleteReq{
			ThreadId: threadId,
			Identity: &assistant_service.Identity{
				UserId: userId,
				OrgId:  orgId,
			},
		})
	}
}

func importSkillIntoWorkspace(ctx *gin.Context, zipURL, skillDir string) (*util.FrontMatter, error) {
	data, err := minio_util.DownloadFileDirect(ctx.Request.Context(), zipURL)
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("download skill zip err: %v", err))
	}
	return importSkillDataIntoWorkspace(data, skillDir)
}

func importSkillDataIntoWorkspace(data []byte, skillDir string) (*util.FrontMatter, error) {
	if err := recreateDir(skillDir); err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("prepare skill dir err: %v", err))
	}
	if err := unzipSkillDir(data, skillDir); err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("unzip skill err: %v", err))
	}
	fm, err := readImportedSkillFrontMatter(skillDir)
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFSkillParse, err.Error())
	}
	return fm, nil
}

func recreateDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

func unzipSkillDir(data []byte, destDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	skillRoot, err := findSkillRootInZip(reader)
	if err != nil {
		return err
	}
	return unzipSkillRoot(reader, skillRoot.zipPath, filepath.Join(destDir, skillRoot.name))
}

func findSkillRootInZip(reader *zip.Reader) (*importedSkillRoot, error) {
	var skillRoots []*importedSkillRoot
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		cleanName, err := cleanZipEntryName(file.Name)
		if err != nil {
			return nil, err
		}
		if path.Base(cleanName) == "SKILL.md" {
			fm, err := parseZipSkillFrontMatter(file)
			if err != nil {
				return nil, err
			}
			skillDirName, err := cleanSkillDirName(fm.Name)
			if err != nil {
				return nil, err
			}
			skillRoots = append(skillRoots, &importedSkillRoot{
				zipPath: path.Dir(cleanName),
				name:    skillDirName,
			})
		}
	}
	switch len(skillRoots) {
	case 0:
		return nil, fmt.Errorf("SKILL.md file not found in the zip archive")
	case 1:
		return skillRoots[0], nil
	default:
		return nil, fmt.Errorf("expected exactly one SKILL.md in the zip archive, got %d", len(skillRoots))
	}
}

func parseZipSkillFrontMatter(file *zip.File) (*util.FrontMatter, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open SKILL.md file: %v", err)
	}
	defer func() { _ = rc.Close() }()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md file: %v", err)
	}
	return util.ParseSkillFrontMatter(string(content))
}

func unzipSkillRoot(reader *zip.Reader, skillRoot, destDir string) error {
	cleanDest, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	cleanDestWithSep := cleanDest + string(os.PathSeparator)

	for _, file := range reader.File {
		cleanName, err := cleanZipEntryName(file.Name)
		if err != nil {
			return err
		}
		if !isZipEntryInSkillRoot(cleanName, skillRoot) {
			continue
		}
		relativeName := relativeZipEntryName(cleanName, skillRoot)
		if relativeName == "." {
			continue
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("zip entry symlink is not supported: %s", file.Name)
		}

		targetPath := filepath.Join(cleanDest, filepath.FromSlash(relativeName))
		absTargetPath, err := filepath.Abs(targetPath)
		if err != nil {
			return err
		}
		if absTargetPath != cleanDest && !strings.HasPrefix(absTargetPath, cleanDestWithSep) {
			return fmt.Errorf("zip entry escapes target directory: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(absTargetPath, safeDirPerm(file.Mode())); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(absTargetPath), 0755); err != nil {
			return err
		}
		if err := writeZipFile(file, absTargetPath); err != nil {
			return err
		}
	}
	return nil
}

func cleanZipEntryName(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." {
			continue
		}
		if part == ".." || strings.Contains(part, ":") {
			return "", fmt.Errorf("invalid zip entry path: %s", name)
		}
	}
	cleanName := path.Clean(name)
	if cleanName == "." || path.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, "../") {
		return "", fmt.Errorf("invalid zip entry path: %s", name)
	}
	return cleanName, nil
}

func isZipEntryInSkillRoot(cleanName, skillRoot string) bool {
	if skillRoot == "." {
		return true
	}
	return cleanName == skillRoot || strings.HasPrefix(cleanName, skillRoot+"/")
}

func relativeZipEntryName(cleanName, skillRoot string) string {
	if skillRoot == "." {
		return cleanName
	}
	if cleanName == skillRoot {
		return "."
	}
	return strings.TrimPrefix(cleanName, skillRoot+"/")
}

func writeZipFile(file *zip.File, targetPath string) error {
	source, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, safeFilePerm(file.Mode()))
	if err != nil {
		return err
	}
	defer func() { _ = target.Close() }()

	_, err = io.Copy(target, source)
	return err
}

func cleanSkillDirName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("skill name is required")
	}
	if name == "." || name == ".." || path.IsAbs(name) || filepath.IsAbs(name) ||
		strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, ":") {
		return "", fmt.Errorf("invalid skill name: %s", name)
	}
	return name, nil
}

func readImportedSkillFrontMatter(skillDir string) (*util.FrontMatter, error) {
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return nil, err
	}
	var skillDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillMdPath := filepath.Join(skillDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillMdPath); err == nil {
			skillDirs = append(skillDirs, filepath.Join(skillDir, entry.Name()))
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	switch len(skillDirs) {
	case 0:
		return nil, fmt.Errorf("SKILL.md file not found in imported skill directory")
	case 1:
		data, err := os.ReadFile(filepath.Join(skillDirs[0], "SKILL.md"))
		if err != nil {
			return nil, err
		}
		return util.ParseSkillFrontMatter(string(data))
	default:
		return nil, fmt.Errorf("expected exactly one imported skill directory, got %d", len(skillDirs))
	}
}

func safeDirPerm(mode os.FileMode) os.FileMode {
	perm := mode.Perm()
	if perm == 0 {
		return 0755
	}
	return perm
}

func safeFilePerm(mode os.FileMode) os.FileMode {
	perm := mode.Perm()
	if perm == 0 {
		return 0644
	}
	return perm
}

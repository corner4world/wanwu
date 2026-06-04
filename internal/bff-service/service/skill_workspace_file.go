package service

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
)

// buildFileTree 递归构建文件树，跳过隐藏文件。
func buildFileTree(basePath, currentPath string, depth int, count *int) ([]*response.FileNode, error) {
	if depth > maxFileTreeDepth || *count >= maxFileTreeNodes {
		return []*response.FileNode{}, nil
	}
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	nodes := make([]*response.FileNode, 0)
	for _, entry := range entries {
		if *count >= maxFileTreeNodes {
			break
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		fullPath := filepath.Join(currentPath, entry.Name())
		relPath, err := filepath.Rel(basePath, fullPath)
		if err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		node := &response.FileNode{
			Name:    entry.Name(),
			Path:    filepath.ToSlash(relPath),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().UnixMilli(),
		}
		*count = *count + 1
		if entry.IsDir() {
			children, err := buildFileTree(basePath, fullPath, depth+1, count)
			if err == nil {
				node.Children = children
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// GetSkillWorkspaceFiles 获取工作区文件树。
func GetSkillWorkspaceFiles(ctx *gin.Context, userId, orgId string, req request.GetSkillWorkspaceFilesReq) (*response.SkillWorkspaceFilesResp, error) {
	log.Infof("[Workspace] GetSkillWorkspaceFiles customSkillID: %v", req.CustomSkillID)

	ws, err := resolveSkillWorkspace(ctx, userId, orgId, req.CustomSkillID)
	if err != nil {
		return nil, err
	}
	if err := ensureNoSymlinkInPath(ws.workspaceDir, ws.workspaceDir, true); err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if info, err := os.Stat(ws.workspaceDir); err != nil {
		if !os.IsNotExist(err) {
			log.Errorf("[Workspace] stat workspace %s error: %v", ws.workspaceDir, err)
		}
		return &response.SkillWorkspaceFilesResp{Files: []*response.FileNode{}}, nil
	} else if !info.IsDir() {
		return &response.SkillWorkspaceFilesResp{Files: []*response.FileNode{}}, nil
	}

	count := 0
	files, err := buildFileTree(ws.workspaceDir, ws.workspaceDir, 0, &count)
	if err != nil {
		log.Errorf("[Workspace] buildFileTree error: %v", err)
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_build_file_tree_failed")
	}
	return &response.SkillWorkspaceFilesResp{Files: files}, nil
}

// GetSkillWorkspaceFile 读取工作区文件内容。
func GetSkillWorkspaceFile(ctx *gin.Context, userId, orgId string, req request.GetSkillWorkspaceFileReq) (*response.SkillWorkspaceFileResp, error) {
	ws, err := resolveSkillWorkspace(ctx, userId, orgId, req.CustomSkillID)
	if err != nil {
		return nil, err
	}

	fullPath, _, err := workspaceFilePath(ws.workspaceDir, req.Path)
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}

	if err := ensureNoSymlinkInPath(ws.workspaceDir, fullPath, true); err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	info, err := os.Lstat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_file_not_found")
		}
		log.Errorf("[Workspace] GetFile stat %s err: %v", fullPath, err)
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_stat_file_failed")
	}
	if info.IsDir() {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_path_is_directory")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "symlink path not allowed")
	}
	if info.Size() > maxFileSize {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_file_too_large")
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		log.Errorf("[Workspace] GetFile read %s err: %v", fullPath, err)
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_read_file_failed")
	}
	return &response.SkillWorkspaceFileResp{
		Content: string(content),
		Size:    info.Size(),
		ModTime: info.ModTime().UnixMilli(),
	}, nil
}

// DownloadSkillWorkspace 下载工作区文件或目录。
func DownloadSkillWorkspace(ctx *gin.Context, userId, orgId string, req request.DownloadSkillWorkspaceReq) (string, []byte, error) {
	ws, err := resolveSkillWorkspace(ctx, userId, orgId, req.CustomSkillID)
	if err != nil {
		return "", nil, err
	}

	fullPath, cleanRelPath, err := workspaceFilePath(ws.workspaceDir, req.Path)
	if err != nil {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if err := ensureNoSymlinkInPath(ws.workspaceDir, fullPath, true); err != nil {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}

	info, err := os.Lstat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_file_not_found")
		}
		log.Errorf("[Workspace] Download stat %s err: %v", fullPath, err)
		return "", nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_stat_file_failed")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "symlink path not allowed")
	}
	if info.IsDir() {
		data, err := util.ZipDir(filepath.Join(fullPath, "."))
		if err != nil {
			log.Errorf("[Workspace] Download zip %s err: %v", fullPath, err)
			return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to create zip: %v", err))
		}
		return fmt.Sprintf("workspace_%s_%s.zip", req.CustomSkillID, filepath.Base(cleanRelPath)), data, nil
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		log.Errorf("[Workspace] Download read %s err: %v", fullPath, err)
		return "", nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_read_file_failed")
	}
	return filepath.Base(cleanRelPath), data, nil
}

// DeleteSkillWorkspaceFile 删除工作区文件或目录。
func DeleteSkillWorkspaceFile(ctx *gin.Context, userId, orgId string, req request.DeleteSkillWorkspaceFileReq) error {
	log.Infof("[Workspace] DeleteFile user=%s org=%s skill=%s path=%s", userId, orgId, req.CustomSkillID, req.Path)
	ws, err := resolveSkillWorkspace(ctx, userId, orgId, req.CustomSkillID)
	if err != nil {
		return err
	}

	fullPath, _, err := workspaceFilePath(ws.workspaceDir, req.Path)
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if err := ensureNoSymlinkInPath(ws.workspaceDir, fullPath, true); err != nil {
		return grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}

	info, err := os.Lstat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_file_not_found")
		}
		log.Errorf("[Workspace] DeleteFile stat %s err: %v", fullPath, err)
		return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_stat_file_failed")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return grpc_util.ErrorStatus(errs.Code_BFFGeneral, "symlink path not allowed")
	}

	if info.IsDir() {
		if err := os.RemoveAll(fullPath); err != nil {
			log.Errorf("[Workspace] DeleteFile remove dir %s err: %v", fullPath, err)
			return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_delete_file_failed")
		}
		return nil
	}
	if err := os.Remove(fullPath); err != nil {
		log.Errorf("[Workspace] DeleteFile remove file %s err: %v", fullPath, err)
		return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_delete_file_failed")
	}
	return nil
}

// UpdateSkillWorkspaceFile 更新工作区文件内容。
func UpdateSkillWorkspaceFile(ctx *gin.Context, userId, orgId string, req request.UpdateSkillWorkspaceFileReq) (*response.UpdateSkillWorkspaceFileResp, error) {
	log.Infof("[Workspace] UpdateFile user=%s org=%s skill=%s path=%s size=%d", userId, orgId, req.CustomSkillID, req.Path, len(req.Content))
	ws, err := resolveSkillWorkspace(ctx, userId, orgId, req.CustomSkillID)
	if err != nil {
		return nil, err
	}

	fullPath, _, err := workspaceFilePath(ws.workspaceDir, req.Path)
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}

	if err := ensureNoSymlinkInPath(ws.workspaceDir, ws.workspaceDir, true); err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if err := os.MkdirAll(ws.workspaceDir, 0755); err != nil {
		log.Errorf("[Workspace] UpdateFile mkdir workspace %s err: %v", ws.workspaceDir, err)
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_create_dir_failed")
	}
	if err := ensureNoSymlinkInPath(ws.workspaceDir, filepath.Dir(fullPath), true); err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		log.Errorf("[Workspace] UpdateFile mkdir %s err: %v", filepath.Dir(fullPath), err)
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_create_dir_failed")
	}
	if err := ensureNoSymlinkInPath(ws.workspaceDir, filepath.Dir(fullPath), true); err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if info, err := os.Lstat(fullPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "symlink path not allowed")
		}
		if info.IsDir() {
			return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_path_is_directory")
		}
	}
	if err := writeFileAtomic(fullPath, []byte(req.Content)); err != nil {
		log.Errorf("[Workspace] UpdateFile write %s err: %v", fullPath, err)
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_write_file_failed")
	}

	return &response.UpdateSkillWorkspaceFileResp{}, nil
}

// matchPattern 检查路径是否匹配 glob 模式（逗号分隔多模式）。
func matchPattern(relPath, pattern string) bool {
	if pattern == "" {
		return true
	}
	for _, p := range strings.Split(pattern, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if matched, err := path.Match(p, filepath.Base(relPath)); err == nil && matched {
			return true
		}
		if matched, err := path.Match(p, relPath); err == nil && matched {
			return true
		}
	}
	return false
}

// writeFileAtomic 通过临时文件和重命名原子写入文件内容。
func writeFileAtomic(filePath string, data []byte) error {
	dir := filepath.Dir(filePath)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpName, filePath); err != nil {
		if removeErr := os.Remove(filePath); removeErr != nil && !os.IsNotExist(removeErr) {
			return err
		}
		if renameErr := os.Rename(tmpName, filePath); renameErr != nil {
			return err
		}
	}
	return nil
}

// shouldIncludeFile 根据 include/exclude 模式决定是否搜索该文件。
func shouldIncludeFile(relPath string, includePattern, excludePattern string) bool {
	if excludePattern != "" && matchPattern(relPath, excludePattern) {
		return false
	}
	if includePattern != "" {
		return matchPattern(relPath, includePattern)
	}
	return true
}

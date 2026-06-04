package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	git_util "github.com/UnicomAI/wanwu/pkg/git-util"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	path_util "github.com/UnicomAI/wanwu/pkg/path-util"
	"github.com/gin-gonic/gin"
)

const (
	maxFileSize                       = 1 * 1024 * 1024 // 最大文件大小 1MB
	maxSearchResults                  = 1000            // 最大搜索结果数
	generalAgentSkillWorkspaceDirName = "skill"         // skill 工作区目录名（与 wga_skill_import.go 中保持一致）
	maxFileTreeNodes                  = 5000
	maxFileTreeDepth                  = 100
	maxSearchLineBytes                = 256 * 1024
	maxSearchResultContentLength      = 4096
)

type skillWorkspaceContext struct {
	customSkillID string
	skillDir      string
	workspaceDir  string
	repo          git_util.Repo
}

// resolveSkillWorkspace 校验归属并解析 Skill 工作区上下文。
func resolveSkillWorkspace(ctx *gin.Context, userId, orgId, customSkillID string) (*skillWorkspaceContext, error) {
	if err := checkCustomSkillOwnership(ctx, userId, orgId, customSkillID); err != nil {
		return nil, err
	}
	skillDir, err := getSkillDir(customSkillID)
	if err != nil {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_get_dir_failed")
	}
	if skillDir == "" {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_not_found")
	}
	return &skillWorkspaceContext{
		customSkillID: customSkillID,
		skillDir:      skillDir,
		workspaceDir:  filepath.Join(skillDir, generalAgentSkillWorkspaceDirName),
		repo:          git_util.Open(skillDir),
	}, nil
}

// resolveInitializedSkillWorkspace 解析并确保工作区已初始化 Git 仓库。
func resolveInitializedSkillWorkspace(ctx *gin.Context, userId, orgId, customSkillID string) (*skillWorkspaceContext, error) {
	ws, err := resolveSkillWorkspace(ctx, userId, orgId, customSkillID)
	if err != nil {
		return nil, err
	}
	if _, err := ensureGitInitializedAt(customSkillID, ws.skillDir); err != nil {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_init_git_failed")
	}
	if !ws.repo.IsInitialized() {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_skill_workspace_git_not_initialized")
	}
	return ws, nil
}


// getSkillDir 获取 skill 目录路径（包含 .git 和 skill/）。
func getSkillDir(customSkillID string) (string, error) {
	store, err := NewGeneralAgentSkillWorkspaceStore(customSkillID)
	if err != nil {
		return "", err
	}
	info := store.GetThreadDir()
	if info.Dir == "" {
		return "", nil
	}
	return info.Dir, nil
}


// ensureGitInitializedAt 在指定 skill 目录上按需初始化 Git 仓库。
func ensureGitInitializedAt(customSkillID, skillDir string) (bool, error) {
	if skillDir == "" {
		return false, nil
	}

	repo := git_util.Open(skillDir)

	// 快速路径：已初始化无需加锁
	if repo.IsInitialized() {
		return false, nil
	}

	// 慢速路径：持仓库锁，防止多请求并发初始化
	mu := repo.GetMutex()
	mu.Lock()
	defer mu.Unlock()

	// 再次检查（另一个 goroutine 可能刚刚完成初始化）
	if repo.IsInitialized() {
		return false, nil
	}

	log.Infof("[Workspace] initializing git repo for skill %s", customSkillID)

	if err := repo.InitLocked(); err != nil {
		return false, fmt.Errorf("git init failed: %w", err)
	}

	return true, nil
}


// workspaceFilePath 解析工作区内文件路径并返回绝对路径与清理后的相对路径。
func workspaceFilePath(basePath, relativePath string) (string, string, error) {
	return path_util.JoinWithinBase(basePath, relativePath, false)
}

// ensureNoSymlinkInPath 校验目标路径链路中不包含符号链接。
func ensureNoSymlinkInPath(basePath, targetPath string, includeTarget bool) error {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return err
	}
	if err := ensureNoSymlinkInAncestors(absBase, true); err != nil {
		return err
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return err
	}
	relSlash := filepath.ToSlash(rel)
	if relSlash == "." {
		return nil
	}
	if relSlash == ".." || strings.HasPrefix(relSlash, "../") || filepath.IsAbs(rel) {
		return fmt.Errorf("path outside workspace")
	}

	parts := strings.Split(relSlash, "/")
	if !includeTarget && len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}
	current := absBase
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink path not allowed")
		}
	}
	return nil
}

// ensureNoSymlinkInAncestors 校验指定路径及其父级不包含符号链接。
func ensureNoSymlinkInAncestors(absPath string, includePath bool) error {
	current := absPath
	if !includePath {
		current = filepath.Dir(current)
	}
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink path not allowed")
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

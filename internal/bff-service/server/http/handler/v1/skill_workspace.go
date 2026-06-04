package v1

import (
	"net/http"
	"net/url"

	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// GetSkillWorkspaceFiles 获取 Skill 工作区文件列表。
//
//	@Tags			resource.skill
//	@Summary		获取Skill工作区文件列表
//	@Description	获取Skill工作区文件树结构
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Success		200				{object}	response.Response{data=response.SkillWorkspaceFilesResp}
//	@Router			/agent/skill/workspace/files [get]
func GetSkillWorkspaceFiles(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GetSkillWorkspaceFilesReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetSkillWorkspaceFiles(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// GetSkillWorkspaceFile 读取 Skill 工作区文件内容。
//
//	@Tags			resource.skill
//	@Summary		读取Skill工作区文件内容
//	@Description	读取指定文件的内容
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Param			path			query		string	true	"文件路径（相对于workspace根目录）"
//	@Success		200				{object}	response.Response{data=response.SkillWorkspaceFileResp}
//	@Router			/agent/skill/workspace/file [get]
func GetSkillWorkspaceFile(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GetSkillWorkspaceFileReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetSkillWorkspaceFile(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// DownloadSkillWorkspace
//
//	@Tags			resource.skill
//	@Summary		下载Skill工作区文件或目录
//	@Description	下载指定工作区文件；目录会打包为 zip
//	@Security		JWT
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Param			path			query		string	true	"文件路径（相对workspace根目录）"
//	@Success		200				{file}		stream
//	@Router			/agent/skill/workspace/download [get]
func DownloadSkillWorkspace(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.DownloadSkillWorkspaceReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	fileName, data, err := service.DownloadSkillWorkspace(ctx, userId, orgId, req)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Data(http.StatusOK, "application/octet-stream", data)
}

// UpdateSkillWorkspaceFile 更新 Skill 工作区文件内容。
//
//	@Tags			resource.skill
//	@Summary		更新Skill工作区文件内容
//	@Description	保存文件修改
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.UpdateSkillWorkspaceFileReq	true	"文件内容"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/workspace/file [put]
func UpdateSkillWorkspaceFile(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.UpdateSkillWorkspaceFileReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.UpdateSkillWorkspaceFile(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// DeleteSkillWorkspaceFile
//
//	@Tags			resource.skill
//	@Summary		删除Skill工作区文件或目录
//	@Description	删除指定工作区文件或目录
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Param			path			query		string	true	"文件路径（相对workspace根目录）"
//	@Success		200				{object}	response.Response
//	@Router			/agent/skill/workspace/file [delete]
func DeleteSkillWorkspaceFile(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.DeleteSkillWorkspaceFileReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	err := service.DeleteSkillWorkspaceFile(ctx, userId, orgId, req)
	gin_util.Response(ctx, nil, err)
}

// SearchSkillWorkspace 搜索 Skill 工作区文件内容。
//
//	@Tags			resource.skill
//	@Summary		搜索Skill工作区内容
//	@Description	在工作区文件中搜索关键词
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.SearchSkillWorkspaceReq	true	"搜索参数"
//	@Success		200		{object}	response.Response{data=response.SkillWorkspaceSearchResp}
//	@Router			/agent/skill/workspace/search [post]
func SearchSkillWorkspace(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.SearchSkillWorkspaceReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.SearchInWorkspace(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// GetSkillWorkspaceGitLog 获取 Skill 工作区 Git 提交历史。
//
//	@Tags			resource.skill
//	@Summary		获取Skill工作区Git提交历史
//	@Description	获取工作区的git commit历史记录
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Param			count			query		int		false	"获取数量（默认50）"
//	@Success		200				{object}	response.Response{data=response.SkillWorkspaceGitLogResp}
//	@Router			/agent/skill/workspace/git/log [get]
func GetSkillWorkspaceGitLog(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GetSkillWorkspaceGitLogReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetSkillWorkspaceGitLog(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// GetSkillWorkspaceGitDiff 获取 Skill 工作区 Git diff。
//
//	@Tags			resource.skill
//	@Summary		获取Skill工作区Git diff
//	@Description	获取两个commit之间的差异
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Param			fromCommit		query		string	false	"起始commit（默认HEAD~1）"
//	@Param			toCommit		query		string	false	"结束commit（默认HEAD）"
//	@Success		200				{object}	response.Response{data=response.SkillWorkspaceGitDiffResp}
//	@Router			/agent/skill/workspace/git/diff [get]
func GetSkillWorkspaceGitDiff(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GetSkillWorkspaceGitDiffReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetSkillWorkspaceGitDiff(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// GetSkillWorkspaceGitFile 获取 Skill 工作区 Git 历史文件内容。
//
//	@Tags			resource.skill
//	@Summary		获取Skill工作区Git历史文件内容
//	@Description	获取指定commit中某文件的内容
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Param			commitHash		query		string	false	"commit hash（默认HEAD）"
//	@Param			filePath		query		string	true	"文件路径"
//	@Success		200				{object}	response.Response{data=response.SkillWorkspaceGitFileResp}
//	@Router			/agent/skill/workspace/git/file [get]
func GetSkillWorkspaceGitFile(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GetSkillWorkspaceGitFileReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetSkillWorkspaceGitFile(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// GetSkillWorkspaceGitFileDiff 获取 Skill 工作区 Git 单文件 diff。
//
//	@Tags			resource.skill
//	@Summary		获取Skill工作区Git单文件diff
//	@Description	获取单个文件在两个commit之间的差异
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Param			fromCommit		query		string	false	"起始commit（默认HEAD~1）"
//	@Param			toCommit		query		string	false	"结束commit（默认HEAD）"
//	@Param			filePath		query		string	true	"文件路径"
//	@Success		200				{object}	response.Response{data=response.SkillWorkspaceGitDiffResp}
//	@Router			/agent/skill/workspace/git/file-diff [get]
func GetSkillWorkspaceGitFileDiff(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GetSkillWorkspaceGitFileDiffReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetSkillWorkspaceGitFileDiff(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// GetGitStatus 获取 Skill 工作区 Git 状态。
//
//	@Tags			resource.skill
//	@Summary		获取Skill工作区Git状态
//	@Description	获取工作区已暂存和未暂存的文件列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Success		200				{object}	response.Response{data=response.GitStatusResp}
//	@Router			/agent/skill/workspace/git/status [get]
func GetGitStatus(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GitStatusReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetGitStatus(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// GitAdd 暂存 Skill 工作区文件。
//
//	@Tags			resource.skill
//	@Summary		暂存Skill工作区文件
//	@Description	将指定文件添加到暂存区（paths为空时暂存全部）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.GitAddReq	true	"暂存参数"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/workspace/git/add [post]
func GitAdd(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GitAddReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.GitAdd(ctx, userId, orgId, req)
	gin_util.Response(ctx, nil, err)
}

// GitReset 取消暂存 Skill 工作区文件。
//
//	@Tags			resource.skill
//	@Summary		取消暂存Skill工作区文件
//	@Description	将指定文件从暂存区移出
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.GitResetReq	true	"取消暂存参数"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/workspace/git/reset [post]
func GitReset(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GitResetReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.GitReset(ctx, userId, orgId, req)
	gin_util.Response(ctx, nil, err)
}

// GitDiscardWorkingTree
//
//	@Tags			resource.skill
//	@Summary		放弃Skill工作区未暂存更改
//	@Description	放弃指定文件的未暂存更改；paths为空时放弃全部未暂存更改
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.GitDiscardReq	true	"放弃更改参数"
//	@Success		200		{object}	response.Response
//	@Router			/agent/skill/workspace/git/discard [post]
func GitDiscardWorkingTree(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GitDiscardReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.GitDiscardWorkingTree(ctx, userId, orgId, req)
	gin_util.Response(ctx, nil, err)
}

// GitCommit 提交 Skill 工作区已暂存变更。
//
//	@Tags			resource.skill
//	@Summary		提交Skill工作区已暂存的变更
//	@Description	提交暂存区的变更（不执行git add）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.GitCommitReq	true	"提交参数"
//	@Success		200		{object}	response.Response{data=string}
//	@Router			/agent/skill/workspace/git/commit [post]
func GitCommit(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GitCommitReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	commitHash, err := service.GitCommitAction(ctx, userId, orgId, req)
	gin_util.Response(ctx, commitHash, err)
}

// GetGitDiffWorkingTree 获取 Skill 工作区未暂存 diff。
//
//	@Tags			resource.skill
//	@Summary		获取Skill工作区未暂存diff
//	@Description	获取工作区与暂存区之间的差异；传 filePath 时返回该文件 index 与 working tree 的全文
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Param			filePath		query		string	false	"文件路径（可选，限制到单个文件）"
//	@Success		200				{object}	response.Response{data=response.SkillWorkspaceGitDiffResp}
//	@Router			/agent/skill/workspace/git/diff-working [get]
func GetGitDiffWorkingTree(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GitStatusReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetGitDiffWorkingTree(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

// GetGitDiffStaged 获取 Skill 工作区已暂存 diff。
//
//	@Tags			resource.skill
//	@Summary		获取Skill工作区已暂存diff
//	@Description	获取暂存区与HEAD之间的差异；传 filePath 时返回该文件 HEAD 与 index 的全文
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			customSkillId	query		string	true	"Skill ID"
//	@Param			filePath		query		string	false	"文件路径（可选，限制到单个文件）"
//	@Success		200				{object}	response.Response{data=response.SkillWorkspaceGitDiffResp}
//	@Router			/agent/skill/workspace/git/diff-staged [get]
func GetGitDiffStaged(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.GitStatusReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetGitDiffStaged(ctx, userId, orgId, req)
	gin_util.Response(ctx, resp, err)
}

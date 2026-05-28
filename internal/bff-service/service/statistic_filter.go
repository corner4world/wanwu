package service

import (
	"slices"

	iam_service "github.com/UnicomAI/wanwu/api/proto/iam-service"
	model_service "github.com/UnicomAI/wanwu/api/proto/model-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	mp "github.com/UnicomAI/wanwu/pkg/model-provider"
	"github.com/gin-gonic/gin"
)

// StatisticViewer 统计模块当前请求者的权限视角（来自 JWT + IAM isAdmin/isSystem）。
type StatisticViewer struct {
	CanUseGlobal  bool // 组织或系统管理员（isAdmin）；为 true 才可在 body 中扩大 orgIds/userIds
	IsSystemAdmin bool // 系统组织管理员；orgIds=ALL 时展开为一级组织及下级
}

func newStatisticViewer(canUseGlobal, isSystemAdmin bool) StatisticViewer {
	return StatisticViewer{
		CanUseGlobal:  canUseGlobal,
		IsSystemAdmin: isSystemAdmin,
	}
}

func GetOrgsStatisticSelect(ctx *gin.Context, viewerUserID, viewerOrgID string, canUseGlobal, isSystemAdmin bool) (*response.ListResult, error) {
	viewer := newStatisticViewer(canUseGlobal, isSystemAdmin)

	if !viewer.CanUseGlobal {
		items := []response.IDName{}
		if viewerOrgID == "" {
			return &response.ListResult{List: items, Total: int64(len(items))}, nil
		}
		org, err := iam.GetOrgInfo(ctx.Request.Context(), &iam_service.GetOrgInfoReq{OrgId: viewerOrgID})
		if err != nil {
			return nil, err
		}
		items = append(items, response.IDName{ID: org.OrgId, Name: org.Name})
		return &response.ListResult{List: items, Total: int64(len(items))}, nil
	}

	if viewer.IsSystemAdmin {
		systemItems, err := collectStatisticSystemAdminOrgItems(ctx, viewerUserID, viewerOrgID)
		if err != nil {
			return nil, err
		}
		return &response.ListResult{List: systemItems, Total: int64(len(systemItems))}, nil
	}

	items, err := collectStatisticOrgAdminOrgItems(ctx, viewerUserID, viewerOrgID)
	if err != nil {
		return nil, err
	}
	return &response.ListResult{List: items, Total: int64(len(items))}, nil
}

// GetUsersStatisticSelect 用户下拉（组织→用户级联第二步）。
// 组织/系统管理员：固定以 JWT orgId 为根，列出该组织及全部下级组织下的用户；忽略 body.orgIds/userIds。
// 普通用户：仅返回本人。
func GetUsersStatisticSelect(ctx *gin.Context, viewerUserID, viewerOrgID string, canUseGlobal bool) (*response.ListResult, error) {
	items := []response.BriefUserInfo{}

	if !canUseGlobal {
		user, err := iam.GetUserInfo(ctx.Request.Context(), &iam_service.GetUserInfoReq{
			UserId: viewerUserID,
			OrgId:  viewerOrgID,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, response.BriefUserInfo{
			UserID:   user.UserId,
			Username: user.UserName,
		})
		return &response.ListResult{List: items, Total: int64(len(items))}, nil
	}

	orgScope, err := collectStatisticOrgAdminOrgIDs(ctx, viewerUserID, viewerOrgID)
	if err != nil {
		return nil, err
	}
	orgScope = fallbackStatisticOrgIDs(orgScope, viewerOrgID)
	if len(orgScope) == 0 {
		return &response.ListResult{List: items, Total: int64(len(items))}, nil
	}

	seen := make(map[string]struct{})
	for _, oid := range orgScope {
		users, err := listOrgUsers(ctx, oid)
		if err != nil {
			return nil, err
		}
		for _, user := range users {
			if user.UserId == "" {
				continue
			}
			if _, ok := seen[user.UserId]; ok {
				continue
			}
			seen[user.UserId] = struct{}{}
			items = append(items, response.BriefUserInfo{
				UserID:   user.UserId,
				Username: user.UserName,
			})
		}
	}
	return &response.ListResult{List: items, Total: int64(len(items))}, nil
}

// GetStatisticModelSelect 模型 Tab 下拉（组织→用户→模型级联第三步）。
// filter 语义同统计接口；无 HasExpansion 时等同模型管理列表（仅 JWT 用户+组织下的模型）。
func GetStatisticModelSelect(ctx *gin.Context, modelType, viewerUserID, viewerOrgID string, filter *request.StatisticFilter, canUseGlobal, isSystemAdmin bool) (*response.ListResult, error) {
	viewer := newStatisticViewer(canUseGlobal, isSystemAdmin)
	if err := validateStatisticFilterAccess(ctx, viewer, *filter, viewerUserID, viewerOrgID); err != nil {
		return nil, err
	}

	if modelType == "" {
		modelType = mp.ModelTypeLLM
	}

	scope, err := ResolveStatisticScope(ctx, *filter, viewerUserID, viewerOrgID, canUseGlobal, isSystemAdmin)
	if err != nil {
		return nil, err
	}

	if !viewer.CanUseGlobal {
		return listModelsLikeModelList(ctx, viewerUserID, viewerOrgID, modelType)
	}
	if len(scope.OrgIds) == 0 || len(scope.UserIds) == 0 {
		return &response.ListResult{List: []any{}, Total: 0}, nil
	}

	merged, err := listModelsInStatisticScope(ctx, scope.OrgIds, scope.UserIds, modelType)
	if err != nil {
		return nil, err
	}
	list, err := toModelInfos(ctx, merged, &ModelInfoOptions{UserId: viewerUserID})
	if err != nil {
		return nil, err
	}
	return &response.ListResult{List: list, Total: int64(len(list))}, nil
}

// listModelsLikeModelList 对齐模型管理列表 GET /model/list（filterScope 为我的模型）
func listModelsLikeModelList(ctx *gin.Context, userID, orgID, modelType string) (*response.ListResult, error) {
	resp, err := model.ListModels(ctx.Request.Context(), &model_service.ListModelsReq{
		UserId:      userID,
		OrgId:       orgID,
		ModelType:   modelType,
		FilterScope: "private",
	})
	if err != nil {
		return nil, err
	}
	list, err := toModelInfos(ctx, resp.Models, &ModelInfoOptions{UserId: userID, OrgId: orgID})
	if err != nil {
		return nil, err
	}
	return &response.ListResult{List: list, Total: int64(len(list))}, nil
}

func listModelsInStatisticScope(ctx *gin.Context, orgIds, userIds []string, modelType string) ([]*model_service.ModelInfo, error) {
	resp, err := model.ListModelsInStatisticScope(ctx.Request.Context(), &model_service.ListModelsInStatisticScopeReq{
		OrgIds:      orgIds,
		UserIds:     userIds,
		ModelType:   modelType,
		FilterScope: "",
	})
	if err != nil {
		return nil, err
	}
	return resp.Models, nil
}

// resolveStatisticGlobalOrgIDs HasOrgExpansion 时的组织 id 列表：ALL 按 IAM 展开，否则原样返回 body.orgIds。
func resolveStatisticGlobalOrgIDs(ctx *gin.Context, filter request.StatisticFilter, viewerUserID, viewerOrgID string, viewer StatisticViewer) ([]string, error) {
	if !slices.Contains(filter.OrgIds, request.StatisticFilterAll) {
		return filter.OrgIds, nil
	}
	var orgIds []string
	var err error
	if viewer.IsSystemAdmin {
		orgIds, err = collectStatisticSystemAdminOrgIDs(ctx, viewerUserID, viewerOrgID)
	} else {
		orgIds, err = collectStatisticOrgAdminOrgIDs(ctx, viewerUserID, viewerOrgID)
	}
	if err != nil {
		return nil, err
	}
	return fallbackStatisticOrgIDs(orgIds, viewerOrgID), nil
}

// resolveStatisticGlobalUserIDs HasUserExpansion 时的用户 id 列表：ALL 为 orgIds 下全部用户，否则原样返回 body.userIds。
func resolveStatisticGlobalUserIDs(ctx *gin.Context, filter request.StatisticFilter, orgIds []string) ([]string, error) {
	if !slices.Contains(filter.UserIds, request.StatisticFilterAll) {
		return filter.UserIds, nil
	}
	return collectStatisticUserIDsInOrgs(ctx, orgIds)
}

// collectStatisticSystemAdminOrgItems 系统组织管理员：一级组织及其全部下级组织
func collectStatisticSystemAdminOrgItems(ctx *gin.Context, viewerUserID, viewerOrgID string) ([]response.IDName, error) {
	resp, err := iam.GetFirstClassOrgAndSubs(ctx.Request.Context(), &iam_service.GetFirstClassOrgAndSubsReq{
		UserId: viewerUserID,
		OrgId:  viewerOrgID,
	})
	if err != nil {
		return nil, err
	}
	return toOrgIDNames(ctx, resp.Orgs, true), nil
}

func collectStatisticSystemAdminOrgIDs(ctx *gin.Context, viewerUserID, viewerOrgID string) ([]string, error) {
	resp, err := iam.GetFirstClassOrgAndSubs(ctx.Request.Context(), &iam_service.GetFirstClassOrgAndSubsReq{
		UserId: viewerUserID,
		OrgId:  viewerOrgID,
	})
	if err != nil {
		return nil, err
	}
	return statisticOrgIDsFromIDNames(resp.Orgs), nil
}

// collectStatisticOrgAdminOrgItems 组织管理员：当前组织及其全部下级组织
func collectStatisticOrgAdminOrgItems(ctx *gin.Context, viewerUserID, viewerOrgID string) ([]response.IDName, error) {
	resp, err := iam.GetOrgAndSubOrgSelectByUser(ctx.Request.Context(), &iam_service.GetOrgAndSubOrgSelectByUserReq{
		UserId: viewerUserID,
		OrgId:  viewerOrgID,
	})
	if err != nil {
		return nil, err
	}
	items := make([]response.IDName, 0, len(resp.Orgs))
	for _, org := range resp.Orgs {
		items = append(items, toOrgIDName(ctx, org))
	}
	return items, nil
}

func collectStatisticOrgAdminOrgIDs(ctx *gin.Context, viewerUserID, viewerOrgID string) ([]string, error) {
	resp, err := iam.GetOrgAndSubOrgSelectByUser(ctx.Request.Context(), &iam_service.GetOrgAndSubOrgSelectByUserReq{
		UserId: viewerUserID,
		OrgId:  viewerOrgID,
	})
	if err != nil {
		return nil, err
	}
	return statisticOrgIDsFromIDNames(resp.Orgs), nil
}

func statisticOrgIDsFromIDNames(orgs []*iam_service.IDName) []string {
	orgIds := make([]string, 0, len(orgs))
	for _, org := range orgs {
		if org != nil && org.Id != "" {
			orgIds = append(orgIds, org.Id)
		}
	}
	return orgIds
}

func listOrgUsers(ctx *gin.Context, orgID string) ([]*iam_service.UserInfo, error) {
	resp, err := iam.GetUserList(ctx.Request.Context(), &iam_service.GetUserListReq{
		OrgId:    orgID,
		PageNo:   -1,
		PageSize: -1,
	})
	if err != nil {
		return nil, err
	}
	return resp.Users, nil
}

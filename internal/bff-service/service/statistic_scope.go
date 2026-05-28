package service

import (
	"slices"

	err_code "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/gin-gonic/gin"
)

// StatisticScope 统计查询解析结果，作为下游 gRPC 的 orgIds、userIds 入参。
type StatisticScope struct {
	OrgIds  []string
	UserIds []string
}

// validateStatisticFilterAccess 校验 body.orgIds/userIds 与 JWT 角色是否匹配。
// - 系统组织管理员：任意 orgIds/userIds（含 ALL，不做 IAM 树校验）
// - 组织管理员：org/user 须在可管理组织树内；ALL 展开范围同下拉接口
// - 普通用户：仅允许空或恰好为 JWT 当前 org/user，禁止 ALL 及他人 id
func validateStatisticFilterAccess(ctx *gin.Context, viewer StatisticViewer, filter request.StatisticFilter, viewerUserID, viewerOrgID string) error {
	if !filter.HasExpansion() {
		return nil
	}
	if viewer.IsSystemAdmin {
		return nil
	}
	if !viewer.CanUseGlobal {
		return validateStatisticFilterSelfOnly(filter, viewerUserID, viewerOrgID)
	}
	return validateStatisticFilterOrgAdmin(ctx, filter, viewerUserID, viewerOrgID)
}

func validateStatisticFilterSelfOnly(filter request.StatisticFilter, viewerUserID, viewerOrgID string) error {
	if slices.Contains(filter.OrgIds, request.StatisticFilterAll) || slices.Contains(filter.UserIds, request.StatisticFilterAll) {
		return grpc_util.ErrorStatus(err_code.Code_BFFInvalidArg, "无权使用组织或用户筛选扩大查询范围")
	}
	for _, oid := range filter.OrgIds {
		if oid != viewerOrgID {
			return grpc_util.ErrorStatus(err_code.Code_BFFInvalidArg, "orgIds 只能为当前组织或留空")
		}
	}
	for _, uid := range filter.UserIds {
		if uid != viewerUserID {
			return grpc_util.ErrorStatus(err_code.Code_BFFInvalidArg, "userIds 只能为当前用户或留空")
		}
	}
	return nil
}

func validateStatisticFilterOrgAdmin(ctx *gin.Context, filter request.StatisticFilter, viewerUserID, viewerOrgID string) error {
	allowedOrgs, err := collectStatisticOrgAdminOrgIDs(ctx, viewerUserID, viewerOrgID)
	if err != nil {
		return err
	}
	allowedOrgs = fallbackStatisticOrgIDs(allowedOrgs, viewerOrgID)
	allowedOrgSet := statisticStringSet(allowedOrgs)

	if slices.Contains(filter.OrgIds, request.StatisticFilterAll) {
		if slices.Contains(filter.UserIds, request.StatisticFilterAll) || len(filter.UserIds) == 0 {
			return nil
		}
		return validateStatisticUserIDsInOrgs(ctx, filter.UserIds, allowedOrgs)
	}

	for _, oid := range filter.OrgIds {
		if _, ok := allowedOrgSet[oid]; !ok {
			return grpc_util.ErrorStatus(err_code.Code_BFFInvalidArg, "orgIds 超出当前管理员可管理范围")
		}
	}

	if slices.Contains(filter.UserIds, request.StatisticFilterAll) || len(filter.UserIds) == 0 {
		return nil
	}
	// userIds 须在 JWT orgId 及全部下级组织内校验，不能仅限定 body.orgIds 范围
	return validateStatisticUserIDsInOrgs(ctx, filter.UserIds, allowedOrgs)
}

func validateStatisticUserIDsInOrgs(ctx *gin.Context, userIds, orgIds []string) error {
	allowedUsers, err := collectStatisticUserIDsInOrgs(ctx, orgIds)
	if err != nil {
		return err
	}
	allowedUserSet := statisticStringSet(allowedUsers)
	for _, uid := range userIds {
		if _, ok := allowedUserSet[uid]; !ok {
			return grpc_util.ErrorStatus(err_code.Code_BFFInvalidArg, "userIds 超出当前管理员可管理范围")
		}
	}
	return nil
}

func statisticStringSet(ids []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			set[id] = struct{}{}
		}
	}
	return set
}

// ResolveStatisticScope 将 body 筛选解析为 orgIds、userIds。
// 无 HasExpansion 或非管理员：固定 JWT 当前 userId+orgId（管理员默认亦如此，不做组织下全员统计）。
// 有 HasExpansion 且为管理员：ALL 走 IAM；具体 id 系统管理员原样下发，组织管理员须已通过 validate。
func ResolveStatisticScope(ctx *gin.Context, filter request.StatisticFilter, viewerUserID, viewerOrgID string, canUseGlobal, isSystemAdmin bool) (StatisticScope, error) {
	viewer := newStatisticViewer(canUseGlobal, isSystemAdmin)
	if err := validateStatisticFilterAccess(ctx, viewer, filter, viewerUserID, viewerOrgID); err != nil {
		return StatisticScope{}, err
	}

	if !viewer.CanUseGlobal || !filter.HasExpansion() {
		return StatisticScope{
			OrgIds:  []string{viewerOrgID},
			UserIds: []string{viewerUserID},
		}, nil
	}

	orgIds, err := resolveStatisticQueryOrgIDs(ctx, filter, viewerUserID, viewerOrgID, viewer)
	if err != nil {
		return StatisticScope{}, err
	}
	userIds, err := resolveStatisticQueryUserIDs(ctx, filter, orgIds)
	if err != nil {
		return StatisticScope{}, err
	}
	if len(orgIds) == 0 || len(userIds) == 0 {
		return StatisticScope{}, grpc_util.ErrorStatus(err_code.Code_BFFInvalidArg, "筛选范围内无可用组织或用户")
	}
	return StatisticScope{OrgIds: orgIds, UserIds: userIds}, nil
}

// fallbackStatisticOrgIDs IAM 展开为空时视为无下级，回退为 JWT 当前组织。
func fallbackStatisticOrgIDs(orgIds []string, viewerOrgID string) []string {
	if len(orgIds) == 0 && viewerOrgID != "" {
		return []string{viewerOrgID}
	}
	return orgIds
}

// resolveStatisticQueryOrgIDs 有 HasOrgExpansion 时解析组织列表，否则为 JWT 当前组织。
func resolveStatisticQueryOrgIDs(ctx *gin.Context, filter request.StatisticFilter, viewerUserID, viewerOrgID string, viewer StatisticViewer) ([]string, error) {
	if filter.HasOrgExpansion() {
		return resolveStatisticGlobalOrgIDs(ctx, filter, viewerUserID, viewerOrgID, viewer)
	}
	return []string{viewerOrgID}, nil
}

// resolveStatisticQueryUserIDs 有 HasUserExpansion 时按 filter 解析用户；否则未传 userIds 时：
// 单组织→该组织全部用户；多组织→上述组织并集的全部用户（内部等价 userIds=ALL）。
func resolveStatisticQueryUserIDs(ctx *gin.Context, filter request.StatisticFilter, orgIds []string) ([]string, error) {
	if filter.HasUserExpansion() {
		return resolveStatisticGlobalUserIDs(ctx, filter, orgIds)
	}
	if len(orgIds) == 1 {
		users, err := listOrgUsers(ctx, orgIds[0])
		if err != nil {
			return nil, err
		}
		userIds := make([]string, 0, len(users))
		for _, u := range users {
			if u.UserId != "" {
				userIds = append(userIds, u.UserId)
			}
		}
		return userIds, nil
	}
	return resolveStatisticGlobalUserIDs(ctx, request.StatisticFilter{UserIds: []string{request.StatisticFilterAll}}, orgIds)
}

// collectStatisticUserIDsInOrgs 给定组织列表下的全部用户 id（去重）。
func collectStatisticUserIDsInOrgs(ctx *gin.Context, orgIds []string) ([]string, error) {
	seen := make(map[string]struct{})
	userIds := make([]string, 0)
	for _, oid := range orgIds {
		if oid == "" {
			continue
		}
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
			userIds = append(userIds, user.UserId)
		}
	}
	return userIds, nil
}

package service

import (
	"strings"

	app_service "github.com/UnicomAI/wanwu/api/proto/app-service"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/pkg/constant"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/gin-gonic/gin"
)

func GetSquareBuiltinSkillList(ctx *gin.Context, userId, orgId, name string) (*response.ListResult, error) {
	skillsCfgList := getSquareSkillConfigs(name)
	list := buildBuiltinSkillInfoList(skillsCfgList)
	return &response.ListResult{
		List:  list,
		Total: int64(len(list)),
	}, nil
}

func GetSquareBuiltinSkillDetail(ctx *gin.Context, skillId string) (*response.BuiltinSkillDetail, error) {
	skillsCfg, exist := config.Cfg().AgentSkill(skillId)
	if !exist {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "skill not found in builtin skills")
	}
	return &response.BuiltinSkillDetail{
		BuiltinSkillInfo: buildBuiltinSkillInfo(skillsCfg),
		SkillMarkdown:    string(skillsCfg.SkillMarkdown),
	}, nil
}

func GetSquareShareSkillList(ctx *gin.Context, userId, orgId, name string) (*response.ListResult, error) {
	appResp, err := app.GetExplorationAppList(ctx.Request.Context(), &app_service.GetExplorationAppListReq{
		Name:       name,
		AppType:    constant.AppTypeSkill,
		SearchType: "all",
		UserId:     userId,
		OrgId:      orgId,
	})
	if err != nil {
		return nil, err
	}
	var skillIds []string
	for _, appInfo := range appResp.GetInfos() {
		if !isSquareShareSkill(appInfo, userId, orgId) {
			continue
		}
		skillIds = append(skillIds, appInfo.GetAppId())
	}
	if len(skillIds) == 0 {
		return &response.ListResult{List: []*response.SharedSkillInfo{}, Total: 0}, nil
	}

	detailResp, err := mcp.GetCustomSkillDetailByIdList(ctx.Request.Context(), &mcp_service.CustomSkillDetailByIdListReq{
		SkillIds: skillIds,
	})
	if err != nil {
		return nil, err
	}
	detailByID := make(map[string]*mcp_service.CustomSkill, len(detailResp.GetSkillDetails()))
	for _, skill := range detailResp.GetSkillDetails() {
		detailByID[skill.GetSkillId()] = skill
	}

	sharedMap, err := getAcquiredSquareSkillMap(ctx, userId, orgId)
	if err != nil {
		return nil, err
	}
	list := make([]*response.SharedSkillInfo, 0, len(skillIds))
	for _, skillId := range skillIds {
		skill := detailByID[skillId]
		if skill == nil {
			continue
		}
		if name != "" && !strings.Contains(skill.GetName(), name) {
			continue
		}
		list = append(list, &response.SharedSkillInfo{
			SkillBasicInfo: response.SkillBasicInfo{
				SkillId: skill.GetSkillId(),
				Name:    skill.GetName(),
				Avatar:  cacheSkillAvatar(ctx, skill.GetAvatar()),
				Author:  skill.GetAuthor(),
				Desc:    skill.GetDesc(),
			},
			IsShared: sharedMap[skill.GetSkillId()],
		})
	}
	return &response.ListResult{
		List:  list,
		Total: int64(len(list)),
	}, nil
}

func getSquareSkillConfigs(name string) []*config.SkillsConfig {
	var skillsCfgList []*config.SkillsConfig
	for _, skillsCfg := range config.Cfg().AgentSkills {
		if name != "" && !strings.Contains(skillsCfg.Name, name) {
			continue
		}
		skillsCfgList = append(skillsCfgList, skillsCfg)
	}
	return skillsCfgList
}

func buildBuiltinSkillInfoList(skillsCfgList []*config.SkillsConfig) []*response.BuiltinSkillInfo {
	list := make([]*response.BuiltinSkillInfo, 0, len(skillsCfgList))
	for _, skillsCfg := range skillsCfgList {
		info := buildBuiltinSkillInfo(*skillsCfg)
		list = append(list, &info)
	}
	return list
}

func buildBuiltinSkillInfo(skillsCfg config.SkillsConfig) response.BuiltinSkillInfo {
	iconUrl := config.Cfg().DefaultIcon.SkillIcon
	if skillsCfg.Avatar != "" {
		iconUrl = skillsCfg.Avatar
	}
	return response.BuiltinSkillInfo{
		SkillBasicInfo: response.SkillBasicInfo{
			SkillId: skillsCfg.SkillId,
			Name:    skillsCfg.Name,
			Avatar:  request.Avatar{Path: iconUrl},
			Author:  skillsCfg.Author,
			Desc:    skillsCfg.Desc,
		},
	}
}

func ShareSquareSkill(ctx *gin.Context, userId, orgId, skillId string) error {
	return bindAcquiredCustomSkill(ctx, userId, orgId, skillId)
}

func bindAcquiredCustomSkill(ctx *gin.Context, userId, orgId, skillId string) error {
	_, err := mcp.AcquiredSkillCreate(ctx.Request.Context(), &mcp_service.AcquiredSkillCreateReq{
		Identity:      &mcp_service.Identity{UserId: userId, OrgId: orgId},
		CustomSkillId: skillId,
	})
	return err
}

func GetSquareShareSkillDetail(ctx *gin.Context, userId, orgId, skillId string) (*response.SharedSkillDetail, error) {
	markdown, err := getLatestPublishedCustomSkillMarkdown(ctx, skillId)
	if err != nil {
		return nil, err
	}
	skill, _, err := getPublishedCustomSkillForShare(ctx, skillId)
	if err != nil {
		return nil, err
	}
	sharedMap, _ := getAcquiredSquareSkillMap(ctx, userId, orgId)
	return &response.SharedSkillDetail{
		SharedSkillInfo: response.SharedSkillInfo{
			SkillBasicInfo: response.SkillBasicInfo{
				SkillId: skill.GetSkillId(),
				Name:    skill.GetName(),
				Avatar:  cacheSkillAvatar(ctx, skill.GetAvatar()),
				Author:  skill.GetAuthor(),
				Desc:    skill.GetDesc(),
			},
			IsShared: sharedMap[skill.GetSkillId()],
		},
		SkillMarkdown: config.FixFrontMatterFormat(markdown),
	}, nil
}

func DownloadSquareShareSkill(ctx *gin.Context, skillId string) ([]byte, error) {
	objectPath, err := getLatestPublishedCustomSkillObjectPath(ctx, skillId)
	if err != nil {
		return nil, err
	}
	return downloadCustomSkillZip(ctx, objectPath)
}

func getLatestPublishedCustomSkillMarkdown(ctx *gin.Context, skillId string) (string, error) {
	if skillId == "" {
		return "", grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "skillId is required")
	}
	publish, err := getLatestPublishCustomSkill(ctx, skillId)
	if err != nil {
		return "", err
	}
	if publish.GetMarkdown() == "" {
		return "", grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "skill_publish_markdown_not_available", "latest skill markdown is empty")
	}
	return publish.GetMarkdown(), nil
}

func getLatestPublishedCustomSkillObjectPath(ctx *gin.Context, skillId string) (string, error) {
	if skillId == "" {
		return "", grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "skillId is required")
	}
	publish, err := getLatestPublishCustomSkill(ctx, skillId)
	if err != nil {
		return "", err
	}
	if publish.GetObjectPath() == "" {
		return "", grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "skill_publish_package_not_available", "latest skill package objectPath is empty")
	}
	return publish.GetObjectPath(), nil
}

func getPublishedCustomSkillForShare(ctx *gin.Context, skillId string) (*mcp_service.CustomSkill, *mcp_service.PublishCustomSkill, error) {
	publish, err := getLatestPublishCustomSkill(ctx, skillId)
	if err != nil {
		return nil, nil, err
	}
	skill := publish.GetSkill()
	if skill == nil || skill.GetSkillId() == "" {
		return nil, nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "custom skill not found")
	}
	return skill, publish, nil
}

func getAcquiredSquareSkillMap(ctx *gin.Context, userId, orgId string) (map[string]bool, error) {
	ret := make(map[string]bool)
	acquiredResp, err := mcp.AcquiredSkillGetList(ctx.Request.Context(), &mcp_service.AcquiredSkillGetListReq{
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		return nil, err
	}
	for _, skill := range acquiredResp.GetList() {
		if customSkill := skill.GetSkill().GetSkill(); customSkill != nil {
			ret[customSkill.GetSkillId()] = true
		}
	}
	return ret, nil
}

// GetSquareShareSkillVersionList 获取共享skill版本列表
func GetSquareShareSkillVersionList(ctx *gin.Context, skillId string) (*response.ListResult, error) {
	if skillId == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "skillId is required")
	}
	return GetSkillVersionList(ctx, skillId)
}

// isSquareShareSkill 判断 skill 是否应出现在广场「共享」列表。
// 包含：他人全局发布；他人在当前组织的组织内/全局发布；本人在其他组织的全局发布。
// 不包含：本人在当前组织的任意发布；私密发布。
func isSquareShareSkill(appInfo *app_service.ExplorationAppInfo, userId, orgId string) bool {
	if appInfo == nil {
		return false
	}
	publishType := appInfo.GetPublishType()
	if publishType == "" || publishType == constant.AppPublishPrivate {
		return false
	}
	isSelf := appInfo.GetUserId() == userId
	isCurrentOrg := appInfo.GetOrgId() == orgId
	if isSelf && isCurrentOrg {
		return false
	}
	if isSelf {
		return publishType == constant.AppPublishPublic
	}
	if publishType == constant.AppPublishPublic {
		return true
	}
	return publishType == constant.AppPublishOrganization && isCurrentOrg
}

// GetSquareCreatedSkillList 获取我发布的skill列表（当前用户、当前组织下所有已发布 skill，含私密/组织内/全局）
func GetSquareCreatedSkillList(ctx *gin.Context, userId, orgId, name string) (*response.ListResult, error) {
	// 必须用 GetAppList（按 userId+orgId 精确过滤），不能用 GetExplorationAppList：
	// 探索列表 searchType=all 会包含「本人所有组织的 private/public」及他人公开，切换组织后仍会看到其他组织的发布。
	appResp, err := app.GetAppList(ctx.Request.Context(), &app_service.GetAppListReq{
		OrgIds:  []string{orgId},
		UserIds: []string{userId},
		AppType: constant.AppTypeSkill,
	})
	if err != nil {
		return nil, err
	}
	skillIds := make([]string, 0, len(appResp.GetInfos()))
	for _, appInfo := range appResp.GetInfos() {
		if appInfo == nil || appInfo.GetAppId() == "" || appInfo.GetPublishType() == "" {
			continue
		}
		skillIds = append(skillIds, appInfo.GetAppId())
	}
	if len(skillIds) == 0 {
		return &response.ListResult{List: []*response.PublishedSkillInfo{}, Total: 0}, nil
	}

	detailResp, err := mcp.GetCustomSkillDetailByIdList(ctx.Request.Context(), &mcp_service.CustomSkillDetailByIdListReq{
		SkillIds: skillIds,
	})
	if err != nil {
		return nil, err
	}
	publishTypeBySkill := make(map[string]string, len(appResp.GetInfos()))
	for _, appInfo := range appResp.GetInfos() {
		if appInfo != nil && appInfo.GetAppId() != "" {
			publishTypeBySkill[appInfo.GetAppId()] = appInfo.GetPublishType()
		}
	}

	versionBySkill, err := getLatestPublishCustomSkillMap(ctx, skillIds)
	if err != nil {
		log.Errorf("GetSquareCreatedSkillList version list failed, userId=%s orgId=%s err=%v", userId, orgId, err)
	}

	list := make([]*response.PublishedSkillInfo, 0, len(skillIds))
	for _, skill := range detailResp.GetSkillDetails() {
		if skill == nil {
			continue
		}
		if name != "" && !strings.Contains(skill.GetName(), name) {
			continue
		}
		item := &response.PublishedSkillInfo{
			SkillBasicInfo: response.SkillBasicInfo{
				SkillId: skill.GetSkillId(),
				Name:    skill.GetName(),
				Avatar:  cacheSkillAvatar(ctx, skill.GetAvatar()),
				Author:  skill.GetAuthor(),
				Desc:    skill.GetDesc(),
			},
			ThreadID:    skill.GetWgaThreadId(),
			PreviewID:   skill.GetPreviewThreadId(),
			IsPublished: true,
			PublishType: publishTypeBySkill[skill.GetSkillId()],
			Version:     "",
		}
		if publish := versionBySkill[skill.GetSkillId()]; publish != nil {
			item.Version = publish.GetVersion()
		}
		list = append(list, item)
	}
	return &response.ListResult{
		List:  list,
		Total: int64(len(list)),
	}, nil
}

// GetSquareCreatedSkillDetail 获取我发布的skill详情
func GetSquareCreatedSkillDetail(ctx *gin.Context, userId, orgId, customSkillId string) (*response.PublishedSkillDetail, error) {
	if customSkillId == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "customSkillId is required")
	}
	// 验证所有权
	if err := checkCustomSkillOwnership(ctx, userId, orgId, customSkillId); err != nil {
		return nil, err
	}

	// 复用 GetCustomSkill 的逻辑
	detail, err := GetCustomSkill(ctx, userId, orgId, customSkillId)
	if err != nil {
		return nil, err
	}

	// 验证是否已发布
	if !detail.IsPublished {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_published", "skill is not published")
	}

	return detail, nil
}

// checkCustomSkillOwnership 验证 skill 是否属于当前用户
func checkCustomSkillOwnership(ctx *gin.Context, userId, orgId, skillId string) error {
	skillList, err := mcp.CustomSkillGetList(ctx.Request.Context(), &mcp_service.CustomSkillGetListReq{
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		return err
	}
	for _, skill := range skillList.GetList() {
		if customSkill := skill.GetSkill(); customSkill != nil && customSkill.GetSkillId() == skillId {
			return nil
		}
	}
	return grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "skill not found or not owned by current user")
}

// checkCustomSkillPublished 验证 skill 是否已发布
func checkCustomSkillPublished(ctx *gin.Context, skillId string) error {
	appResp, err := app.GetAppListByIds(ctx.Request.Context(), &app_service.GetAppListByIdsReq{
		AppIdsList: []string{skillId},
		AppType:    constant.AppTypeSkill,
	})
	if err != nil {
		return err
	}
	if len(appResp.GetInfos()) == 0 || appResp.GetInfos()[0].GetPublishType() == "" {
		return grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_published", "skill is not published")
	}
	return nil
}

// GetSquareCreatedSkillVersionList 获取我发布skill版本列表
func GetSquareCreatedSkillVersionList(ctx *gin.Context, userId, orgId, customSkillId string) (*response.ListResult, error) {
	if customSkillId == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "customSkillId is required")
	}
	// 验证所有权和发布状态
	if err := checkCustomSkillOwnership(ctx, userId, orgId, customSkillId); err != nil {
		return nil, err
	}
	if err := checkCustomSkillPublished(ctx, customSkillId); err != nil {
		return nil, err
	}
	return GetSkillVersionList(ctx, customSkillId)
}

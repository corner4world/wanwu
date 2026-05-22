package service

import (
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
)

// GetAcquiredSkillList 资源库-我添加的skill列表
func GetAcquiredSkillList(ctx *gin.Context, userId, orgId, name string) (*response.ListResult, error) {
	resp, err := mcp.AcquiredSkillGetList(ctx.Request.Context(), &mcp_service.AcquiredSkillGetListReq{
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
		Name:     name,
	})
	if err != nil {
		return nil, err
	}

	list := make([]*response.AcquiredSkillListItem, 0, len(resp.List))
	for _, skill := range resp.List {
		list = append(list, toAcquiredSkillListItem(ctx, skill))
	}

	return &response.ListResult{
		List:  list,
		Total: resp.Total,
	}, nil
}

// DeleteAcquiredSkill 资源库-删除已添加的skill
func DeleteAcquiredSkill(ctx *gin.Context, userId, orgId, acquiredSkillId string) error {
	if _, err := getOwnedAcquiredSkill(ctx, userId, orgId, acquiredSkillId); err != nil {
		return err
	}
	_, err := mcp.AcquiredSkillDelete(ctx.Request.Context(), &mcp_service.AcquiredSkillDeleteReq{
		AcquiredSkillId: acquiredSkillId,
	})
	return err
}

// GetAcquiredSkill 资源库-获取已添加skill详情
func GetAcquiredSkill(ctx *gin.Context, userId, orgId, acquiredSkillId string) (*response.AcquiredSkillDetail, error) {
	skill, err := getAcquiredSkillByID(ctx, acquiredSkillId)
	if err != nil {
		return nil, err
	}
	detail := toAcquiredSkillDetail(ctx, skill, true)
	return detail, nil
}

func GetCallbackAcquiredSkillListDetail(ctx *gin.Context, acquiredSkillIdList []string) (*response.CallbackAcquiredSkillDetailListResp, error) {
	if len(acquiredSkillIdList) == 0 {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "skillIdList is required")
	}
	filteredAcquiredSkillIdList := make([]string, 0, len(acquiredSkillIdList))
	for _, acquiredSkillId := range acquiredSkillIdList {
		if acquiredSkillId == "" {
			continue
		}
		filteredAcquiredSkillIdList = append(filteredAcquiredSkillIdList, acquiredSkillId)
	}
	if len(filteredAcquiredSkillIdList) == 0 {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "skillIdList is required")
	}

	skillByID, err := getAcquiredSkillByIDMap(ctx, filteredAcquiredSkillIdList)
	if err != nil {
		return nil, err
	}

	skillDetailList := make([]*response.CallbackAcquiredSkillDetail, 0, len(filteredAcquiredSkillIdList))
	for _, acquiredSkillId := range filteredAcquiredSkillIdList {
		skill := skillByID[acquiredSkillId]
		if skill == nil {
			log.Warnf("callback acquired skill list ignored missing acquired skill, acquiredSkillId: %s", acquiredSkillId)
			continue
		}
		detail, err := toCallbackAcquiredSkillDetail(ctx, skill)
		if err != nil {
			log.Warnf("callback acquired skill list ignored invalid acquired skill, acquiredSkillId: %s, err: %v", acquiredSkillId, err)
			continue
		}
		skillDetailList = append(skillDetailList, detail)
	}

	return &response.CallbackAcquiredSkillDetailListResp{SkillList: skillDetailList}, nil
}

func DownloadAcquiredSkill(ctx *gin.Context, userId, orgId, acquiredSkillId string) ([]byte, error) {
	skill, err := getAcquiredSkillByID(ctx, acquiredSkillId)
	if err != nil {
		return nil, err
	}
	if skill.GetSkill().GetObjectPath() == "" {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "skill_publish_package_not_available", "acquired skill package objectPath is empty")
	}
	return downloadCustomSkillZip(ctx, skill.GetSkill().GetObjectPath())
}

func GetAcquiredSkillVersionList(ctx *gin.Context, userId, orgId, acquiredSkillId string) (*response.ListResult, error) {
	sourceSkillId, err := getAcquiredSourceCustomSkillID(ctx, acquiredSkillId)
	if err != nil {
		return nil, err
	}
	return GetSkillVersionList(ctx, sourceSkillId)
}

// GetSkillVersionList 获取 skill 版本列表
func GetSkillVersionList(ctx *gin.Context, skillId string) (*response.ListResult, error) {
	resp, err := mcp.GetPublishCustomSkillHistoryList(ctx.Request.Context(), &mcp_service.GetPublishCustomSkillHistoryListReq{
		SkillId: skillId,
	})
	if err != nil {
		return nil, err
	}
	list := make([]*response.AcquiredSkillVersionInfo, 0, len(resp.GetHistoryList()))
	for _, item := range resp.GetHistoryList() {
		list = append(list, &response.AcquiredSkillVersionInfo{
			Version:   item.GetVersion(),
			Desc:      item.GetVersionDesc(),
			UpdatedAt: util.Time2Str(item.GetCreatedAt()),
		})
	}
	return &response.ListResult{
		List:  list,
		Total: int64(len(list)),
	}, nil
}

func getAcquiredSourceCustomSkillID(ctx *gin.Context, acquiredSkillId string) (string, error) {
	if acquiredSkillId == "" {
		return "", grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "skillId is required")
	}
	skill, err := getAcquiredSkillByID(ctx, acquiredSkillId)
	if err != nil {
		return "", err
	}
	sourceSkillId := skill.GetSkill().GetSkill().GetSkillId()
	if sourceSkillId == "" {
		return "", grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "skill_acquired_source_custom_id_not_available", "acquired skill source custom skill id is empty")
	}
	return sourceSkillId, nil
}

// --- internal ---

func getAcquiredSkillByID(ctx *gin.Context, acquiredSkillId string) (*mcp_service.AcquiredSkill, error) {
	if acquiredSkillId == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "skillId is required")
	}
	return mcp.AcquiredSkillGet(ctx.Request.Context(), &mcp_service.AcquiredSkillGetReq{
		AcquiredSkillId: acquiredSkillId,
	})
}

func getAcquiredSkillByIDMap(ctx *gin.Context, acquiredSkillIdList []string) (map[string]*mcp_service.AcquiredSkill, error) {
	filteredAcquiredSkillIdList := make([]string, 0, len(acquiredSkillIdList))
	for _, acquiredSkillId := range acquiredSkillIdList {
		if acquiredSkillId == "" {
			continue
		}
		filteredAcquiredSkillIdList = append(filteredAcquiredSkillIdList, acquiredSkillId)
	}

	ret := make(map[string]*mcp_service.AcquiredSkill, len(filteredAcquiredSkillIdList))
	if len(filteredAcquiredSkillIdList) == 0 {
		return ret, nil
	}

	resp, err := mcp.AcquiredSkillGetByIDList(ctx.Request.Context(), &mcp_service.AcquiredSkillGetByIDListReq{
		AcquiredSkillIdList: uniqueSkillIDs(filteredAcquiredSkillIdList),
	})
	if err != nil {
		return nil, err
	}

	for _, skill := range resp.GetList() {
		if skill == nil || skill.GetAcquiredSkillId() == "" {
			continue
		}
		ret[skill.GetAcquiredSkillId()] = skill
	}
	return ret, nil
}

func toAcquiredSkillDetail(ctx *gin.Context, skill *mcp_service.AcquiredSkill, includeVariables bool) *response.AcquiredSkillDetail {
	if skill == nil {
		return nil
	}
	publish := skill.GetSkill()
	customSkill := publish.GetSkill()
	ret := &response.AcquiredSkillDetail{
		SkillId:       skill.GetAcquiredSkillId(),
		Name:          customSkill.GetName(),
		Avatar:        cacheSkillAvatar(ctx, customSkill.GetAvatar()),
		Author:        customSkill.GetAuthor(),
		Desc:          customSkill.GetDesc(),
		SkillMarkdown: config.FixFrontMatterFormat(publish.GetMarkdown()),
	}
	if includeVariables {
		variables, err := getAcquiredSkillVariables(ctx, skill.GetAcquiredSkillId())
		if err == nil {
			ret.Variables = toSkillVariables(variables)
		}
	}
	return ret
}

func toCallbackAcquiredSkillDetail(ctx *gin.Context, skill *mcp_service.AcquiredSkill) (*response.CallbackAcquiredSkillDetail, error) {
	if skill == nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "acquired skill not found")
	}
	publish := skill.GetSkill()
	customSkill := publish.GetSkill()
	if customSkill == nil || customSkill.GetSkillId() == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_published", "custom skill is not published")
	}
	if publish.GetObjectPath() == "" {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "skill_publish_package_not_available", "latest skill package objectPath is empty")
	}
	return &response.CallbackAcquiredSkillDetail{
		SkillId:    skill.GetAcquiredSkillId(),
		Name:       customSkill.GetName(),
		Avatar:     cacheSkillAvatar(ctx, customSkill.GetAvatar()),
		Author:     customSkill.GetAuthor(),
		Desc:       customSkill.GetDesc(),
		ObjectPath: publish.GetObjectPath(),
	}, nil
}

func toAcquiredSkillListItem(ctx *gin.Context, skill *mcp_service.AcquiredSkill) *response.AcquiredSkillListItem {
	if skill == nil {
		return nil
	}
	customSkill := skill.GetSkill().GetSkill()
	return &response.AcquiredSkillListItem{
		SkillId: skill.GetAcquiredSkillId(),
		Name:    customSkill.GetName(),
		Avatar:  cacheSkillAvatar(ctx, customSkill.GetAvatar()),
		Author:  customSkill.GetAuthor(),
		Desc:    customSkill.GetDesc(),
	}
}

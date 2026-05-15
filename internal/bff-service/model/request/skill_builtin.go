package request

type SearchBuiltinSkillListReq struct {
	SkillIdList []string `json:"skillIdList" form:"skillIdList" validate:"required"`
	CommonCheck
}

type SearchCustomSkillListReq struct {
	SkillIdList []string `json:"skillIdList" form:"skillIdList" validate:"required"`
	CommonCheck
}

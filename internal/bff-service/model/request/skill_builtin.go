package request

type SearchBuiltinSkillListReq struct {
	SkillIdList []string `json:"skillIdList" form:"skillIdList" validate:"required"`
	CommonCheck
}

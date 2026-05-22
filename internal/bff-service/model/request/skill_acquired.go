package request

type DeleteAcquiredSkillReq struct {
	SkillId string `json:"skillId" validate:"required"`
}

func (r *DeleteAcquiredSkillReq) Check() error {
	return nil
}

type AcquiredSkillIDReq struct {
	SkillId string `form:"skillId" json:"skillId" validate:"required"`
}

func (r *AcquiredSkillIDReq) Check() error {
	return nil
}

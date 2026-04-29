package request

type DeleteAcquiredSkillReq struct {
	SkillId string `json:"skillId" validate:"required"`
}

func (r *DeleteAcquiredSkillReq) Check() error {
	return nil
}

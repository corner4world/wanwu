package request

type DeleteJoinerSkillReq struct {
	SkillId string `json:"skillId" validate:"required"`
}

func (r *DeleteJoinerSkillReq) Check() error {
	return nil
}

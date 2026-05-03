package request

type ShareSquareSkillReq struct {
	SkillId string `json:"skillId" validate:"required"`
}

func (r *ShareSquareSkillReq) Check() error {
	return nil
}

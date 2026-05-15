package request

import "fmt"

type SkillVariable struct {
	Name          string `json:"name"`
	Desc          string `json:"desc"`
	VariableKey   string `json:"variableKey"`
	VariableValue string `json:"variableValue"`
}

type SkillConfigReq struct {
	SkillId  string        `json:"skillId" validate:"required"`
	Variable SkillVariable `json:"variable" validate:"required"`
}

func (r *SkillConfigReq) Check() error {
	if r.Variable.Name == "" {
		return fmt.Errorf("variable.name is required")
	}
	return nil
}

type UpdateSkillConfigReq struct {
	ID       string        `json:"id" validate:"required"`
	Variable SkillVariable `json:"variable" validate:"required"`
}

func (r *UpdateSkillConfigReq) Check() error {
	if r.Variable.Name == "" {
		return fmt.Errorf("variable.name is required")
	}
	return nil
}

type DeleteSkillConfigReq struct {
	ID string `json:"id" validate:"required"`
}

func (r *DeleteSkillConfigReq) Check() error {
	return nil
}

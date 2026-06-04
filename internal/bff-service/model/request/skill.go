package request

import "fmt"

// --- Skill Variable ---

// SkillVariable 变量配置
type SkillVariable struct {
	Name          string `json:"name"`
	Desc          string `json:"desc"`
	VariableKey   string `json:"variableKey"`
	VariableValue string `json:"variableValue"`
}

// --- Custom Skill ---

type CreateCustomSkillReq struct {
	Avatar Avatar `json:"avatar" form:"avatar"`
	Author string `json:"author" form:"author"`
	ZipUrl string `json:"zipUrl" form:"zipUrl" validate:"required"`
}

func (c *CreateCustomSkillReq) Check() error {
	return nil
}

type CustomSkillIDReq struct {
	SkillId string `json:"skillId" validate:"required"`
}

func (c *CustomSkillIDReq) Check() error {
	return nil
}

type DeleteCustomSkillReq struct {
	SkillId string `json:"skillId" validate:"required"`
}

func (c *DeleteCustomSkillReq) Check() error {
	return nil
}

type CheckCustomSkillReq struct {
	ZipUrl string `json:"zipUrl" form:"zipUrl" validate:"required"`
}

func (c *CheckCustomSkillReq) Check() error {
	return nil
}

type CustomSkillVersionDownloadReq struct {
	SkillId string `form:"skillId" json:"skillId" validate:"required"`
	Version string `form:"version" json:"version" validate:"required"`
}

func (c *CustomSkillVersionDownloadReq) Check() error {
	return nil
}

// --- Acquired Skill ---

type DeleteAcquiredSkillReq struct {
	SkillId string `json:"skillId" validate:"required"`
}

func (r *DeleteAcquiredSkillReq) Check() error {
	return nil
}

// --- Square Skill ---

type ShareSquareSkillReq struct {
	SkillId string `json:"skillId" validate:"required"`
}

func (r *ShareSquareSkillReq) Check() error {
	return nil
}

// --- Skill Config (Builtin/Custom/Acquired 通用) ---

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

// --- Callback Skill Search ---

type SearchBuiltinSkillListReq struct {
	SkillIdList []string `json:"skillIdList" form:"skillIdList" validate:"required"`
	CommonCheck
}

type SearchCustomSkillListReq struct {
	SkillIdList []string `json:"skillIdList" form:"skillIdList" validate:"required"`
	CommonCheck
}

type SearchAcquiredSkillListReq struct {
	SkillIdList []string `json:"skillIdList" form:"skillIdList" validate:"required"`
	CommonCheck
}

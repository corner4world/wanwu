package request

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

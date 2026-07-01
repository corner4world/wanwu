package request

type UserCreate struct {
	Username string `json:"username" validate:"required"` // 用户名
	UserInfo
}

func (u *UserCreate) Check() error {
	return nil
}

type UserUpdate struct {
	UserID string `json:"userId" validate:"required"` // 用户ID
	UserInfo
}

func (u *UserUpdate) Check() error {
	return nil
}

type UserInfo struct {
	Nickname string   `json:"nickname"`                     // 昵称
	Cipher   string   `json:"cipher" validate:"required"`   // RSA加密后的Base64字符串，包含{password, challenge}
	Phone    string   `json:"phone"`                        // 电话
	Remark   string   `json:"remark"`                       // 备注
	Gender   string   `json:"gender"`                       // 性别（0-女，1-男，空-未知）
	Company  string   `json:"company"`                      // 公司
	RoleIDs  []string `json:"roleIds" validate:"max=1"`     // 角色列表
	KeyID    string   `json:"keyId" validate:"required"`   // RSA公钥ID
}

type UserID struct {
	UserID string `json:"userId" validate:"required"` // 用户ID
}

func (u *UserID) Check() error {
	return nil
}

type UserStatus struct {
	UserID
	Status bool `json:"status"`
}

func (u *UserStatus) Check() error {
	return nil
}

type UserPassword struct {
	UserID
	OldCipher string `json:"oldCipher" validate:"required"` // 旧密码RSA加密后的Base64字符串，包含{password, challenge}
	NewCipher string `json:"newCipher" validate:"required"` // 新密码RSA加密后的Base64字符串，包含{password, challenge}
	KeyID     string `json:"keyId" validate:"required"`   // RSA公钥ID
}

func (u *UserPassword) Check() error {
	return nil
}

type UserPasswordByAdmin struct {
	UserID
	Cipher string `json:"cipher" validate:"required"`   // RSA加密后的Base64字符串，包含{password, challenge}
	KeyID  string `json:"keyId" validate:"required"`  // RSA公钥ID
}

func (u *UserPasswordByAdmin) Check() error {
	return nil
}

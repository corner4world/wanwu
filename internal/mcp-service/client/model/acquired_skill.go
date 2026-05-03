package model

type AcquiredSkill struct {
	ID                 uint32 `gorm:"primarykey"`
	SquareSkillID      string `gorm:"column:square_skill_id;index:idx_acquired_skill_square_skill_id;comment:skill id"`
	Name               string `gorm:"column:name;index:idx_acquired_skill_name;comment:skill名称"`
	Avatar             string `gorm:"column:avatar;comment:skill头像"`
	Author             string `gorm:"column:author;index:idx_acquired_skill_author;comment:作者"`
	AuthorID           string `gorm:"column:author_id;index:idx_acquired_skill_author_id;comment:作者id"`
	Desc               string `gorm:"column:desc;comment:skill描述"`
	ObjectPath         string `gorm:"column:object_path;comment:skill数据minio对象路径(zip压缩包)"`
	Markdown           string `gorm:"column:markdown;type:text;comment:skill markdown内容"`
	AcquiredType       string `gorm:"column:acquired_type;index:idx_acquired_skill_acquired_type;comment:来源类型(builtin/custom)"`
	Version            string `gorm:"column:version;index:idx_acquired_skill_version;comment:版本号"`
	VersionDescription string `gorm:"column:version_description;type:text;comment:版本描述"`
	UserID             string `gorm:"column:user_id;index:idx_acquired_skill_user_id;comment:用户id"`
	OrgID              string `gorm:"column:org_id;index:idx_acquired_skill_org_id;comment:组织id"`
	CreatedAt          int64  `gorm:"autoCreateTime:milli;comment:创建时间"`
	UpdatedAt          int64  `gorm:"autoUpdateTime:milli;comment:更新时间"`
}

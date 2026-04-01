package model

type WgaConversationConfig struct {
	ID          uint32 `gorm:"column:id;primary_key;type:bigint(20) auto_increment;not null;comment:配置Id"`
	ThreadID    string `gorm:"column:thread_id;uniqueIndex;type:varchar(128);not null;comment:对话ID"`
	ModelConfig string `gorm:"column:model_config;type:json;comment:模型配置JSON"`
	UserID      string `gorm:"column:user_id;index;type:varchar(64);comment:用户id"`
	OrgID       string `gorm:"column:org_id;index;type:varchar(64);comment:组织id"`
	CreatedAt   int64  `gorm:"autoCreateTime:milli;comment:创建时间"`
	UpdatedAt   int64  `gorm:"autoUpdateTime:milli;comment:更新时间"`
}

type WgaConfig struct {
	ID            uint32 `gorm:"column:id;primary_key;type:bigint(20) auto_increment;not null;comment:配置Id"`
	ToolList      string `gorm:"column:tool_list;type:json;comment:工具列表JSON"`
	AssistantList string `gorm:"column:assistant_list;type:json;comment:智能体列表JSON"`
	UserID        string `gorm:"column:user_id;uniqueIndex;type:varchar(64);not null;comment:用户id"`
	OrgID         string `gorm:"column:org_id;index;type:varchar(64);not null;comment:组织id"`
	CreatedAt     int64  `gorm:"autoCreateTime:milli;comment:创建时间"`
	UpdatedAt     int64  `gorm:"autoUpdateTime:milli;comment:更新时间"`
}

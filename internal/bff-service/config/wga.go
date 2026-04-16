package config

import (
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
)

var _wga *WgaConfig

type WgaConfig struct {
	ConfigPath  string               `yaml:"configPath" json:"configPath" mapstructure:"configPath"`
	AgentID     string               `yaml:"agentId" json:"agentId" mapstructure:"agentId"`
	Model       WgaModelConfig       `yaml:"model" json:"model" mapstructure:"model"`
	Persistent  WgaPersistentConfig  `yaml:"persistent" json:"persistent" mapstructure:"persistent"`
	Tools       []WgaToolConfig      `yaml:"tools" json:"tools" mapstructure:"tools"`
	SubAgents   []WgaAgentInfo       `yaml:"sub_agents" json:"sub_agents" mapstructure:"sub_agents"`
	UploadLimit WgaUploadLimitConfig `yaml:"upload_limit" json:"upload_limit" mapstructure:"upload_limit"`
}

type WgaAgentInfo struct {
	AgentID     string `yaml:"agent_id" json:"agent_id" mapstructure:"agent_id"`
	AgentName   string `yaml:"agent_name" json:"agent_name" mapstructure:"agent_name"`
	AvatarPath  string `yaml:"avatar_path" json:"avatar_path" mapstructure:"avatar_path"`
	Placeholder string `yaml:"placeholder" json:"placeholder" mapstructure:"placeholder"`
}

type WgaUploadLimitConfig struct {
	ImageTypes   string `yaml:"image_types" json:"image_types" mapstructure:"image_types"`
	FileTypes    string `yaml:"file_types" json:"file_types" mapstructure:"file_types"`
	MaxImageSize int    `yaml:"max_image_size" json:"max_image_size" mapstructure:"max_image_size"`
	MaxFileSize  int    `yaml:"max_file_size" json:"max_file_size" mapstructure:"max_file_size"`
}

type WgaModelConfig struct {
	Provider     string `yaml:"provider" json:"provider" mapstructure:"provider"`
	ProviderName string `yaml:"provider_name" json:"provider_name" mapstructure:"provider_name"`
	BaseURL      string `yaml:"base_url" json:"base_url" mapstructure:"base_url"`
	APIKey       string `yaml:"api_key" json:"api_key" mapstructure:"api_key"`
	Model        string `yaml:"model" json:"model" mapstructure:"model"`
	ModelName    string `yaml:"model_name" json:"model_name" mapstructure:"model_name"`
}

type WgaPersistentConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled" mapstructure:"enabled"`    // 是否启用持久化
	BaseDir string `yaml:"base_dir" json:"base_dir" mapstructure:"base_dir"` // 持久化根目录
	Mode    string `yaml:"mode" json:"mode" mapstructure:"mode"`             // 持久化模式：overwrite 或 versioned
}

type WgaToolConfig struct {
	Title   string                 `yaml:"title" json:"title" mapstructure:"title"`
	APIAuth util.ApiAuthWebRequest `yaml:"apiAuth" json:"apiAuth" mapstructure:"apiAuth"`
}

func LoadWgaConfig(path string) error {
	_wga = &WgaConfig{}
	return util.LoadConfig(path, _wga)
}

func WgaCfg() *WgaConfig {
	if _wga == nil {
		log.Panicf("wga config not loaded")
	}
	return _wga
}

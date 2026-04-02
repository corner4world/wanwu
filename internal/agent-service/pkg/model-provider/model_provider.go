package model_provider

import (
	"github.com/UnicomAI/wanwu/internal/agent-service/pkg"
	"github.com/UnicomAI/wanwu/internal/agent-service/pkg/config"
	mp "github.com/UnicomAI/wanwu/pkg/model-provider"
)

var modelProvider = ModelProvider{}

type ModelProvider struct {
}

func init() {
	pkg.AddContainer(modelProvider)
}

func (c ModelProvider) LoadType() string {
	return "model-provider"
}

func (c ModelProvider) Load() error {
	mp.Init(config.GetConfig().BffServer.Endpoint)
	return nil
}

func (c ModelProvider) StopPriority() int {
	return pkg.DefaultPriority
}

func (c ModelProvider) Stop() error {
	return nil
}

package tracer

import (
	"github.com/UnicomAI/wanwu/internal/agent-service/pkg"
	trace_util "github.com/UnicomAI/wanwu/pkg/trace-util"
)

var wanWuTracer = WanWuTracer{}

type WanWuTracer struct {
}

func init() {
	pkg.AddContainer(wanWuTracer)
}

func (c WanWuTracer) LoadType() string {
	return "tracer"
}

func (c WanWuTracer) Load() error {
	err := trace_util.InitTracer()
	if err != nil {
		return err
	}
	return nil
}

func (c WanWuTracer) StopPriority() int {
	return pkg.DefaultPriority
}

func (c WanWuTracer) Stop() error {
	return nil
}

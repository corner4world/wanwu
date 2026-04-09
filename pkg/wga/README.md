# WGA Wanwu Generalist Agent

万悟通用智能体统一管理和执行接口，支持多种智能体类型的组合与编排。

## 架构

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                                      应用层                                          │
│                                                                                     │
│    examples/agui-demo/backend-wga           examples/agui-demo/backend-wga-sandbox  │
│    ┌───────────────────────┐              ┌───────────────────────────┐            │
│    │ HandleSSE()           │              │ HandleSSE()               │            │
│    │  ├─ wga.Run()         │              │  ├─ wga_sandbox.Run()     │            │
│    │  ├─ EinoTranslator    │              │  ├─ OpencodeTranslator    │            │
│    │  └─ SSE 响应          │              │  └─ SSE 响应              │            │
│    └───────────────────────┘              └───────────────────────────┘            │
└─────────────────────────────────────────────────────────────────────────────────────┘
                              │                              │
                              ▼                              ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                                        wga                                           │
│                                高级 API - 智能体统一接口                              │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ api.go                                                                        │ │
│  │  ├─ Init(configPath)        加载智能体配置                                     │ │
│  │  ├─ CheckOptions(id, opts)  检查运行条件                                      │ │
│  │  ├─ Run(id, opts)          执行智能体，返回 AgentEvent 迭代器                 │ │
│  │  └─ Cleanup(runID)          清理沙箱资源                                       │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                             │
│                                       ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ internal/factory/agent.go - 智能体工厂                                         │ │
│  │                                                                               │ │
│  │  newAgent()                                                                   │ │
│  │   ├─ AgentTypeReAct      → newReactAgent()      → eino adk                   │ │
│  │   ├─ AgentTypeSandbox    → newSandboxAgent()    → wga-sandbox                │ │
│  │   ├─ AgentTypeSequential → newSequentialAgent() → 组合智能体                  │ │
│  │   ├─ AgentTypeLoop       → newLoopAgent()       → 组合智能体                  │ │
│  │   ├─ AgentTypeParallel   → newParallelAgent()   → 组合智能体                  │ │
│  │   ├─ AgentTypeDeep       → newDeepAgent()       → 深度思考智能体              │ │
│  │   └─ AgentTypeSupervisor → newSupervisorAgent() → 监督者智能体                │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                             │
│                                       ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ internal/factory/agent_sandbox.go - 沙箱智能体                                  │ │
│  │                                                                               │ │
│  │  sandboxAgent.Run()                                                          │ │
│  │   │                                                                           │ │
│  │   │  1. buildSandboxOpts()     构建沙箱选项                                    │ │
│  │   │     ├─ ModelConfig (来自 WithModelConfig)                                 │ │
│  │   │     ├─ Instruction (来自配置文件)                                          │ │
│  │   │     ├─ Tools (来自配置 + WithToolConfig + WithExtraTool)                  │ │
│  │   │     └─ MCPs (来自 WithMCP)                                                │ │
│  │   │                                                                           │ │
│  │   │  2. wga_sandbox.Run()  ─────────────────▶ wga-sandbox (opencode runner)  │ │
│  │   │                                                                           │ │
│  │   │  3. ConvertToEinoIterator() 转换 JSON → AgentEvent                        │ │
│  │   │                                                                           │ │
│  │   │  4. 返回 Iterator                                                          │ │
│  │   ▼                                                                           │ │
│  │  返回: *adk.AsyncIterator[*adk.AgentEvent]                                    │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                             │
│                                       ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ internal/option/ - 运行选项                                                     │ │
│  │                                                                               │ │
│  │  option.go        Options 结构体、选项函数、Check 方法                         │ │
│  │  option_model.go  ToChatModel() 模型转换                                      │ │
│  │  option_prompt.go FormatInstruction() 提示词格式化                             │ │
│  │  option_tool.go   ToToolsConfig() 工具配置转换                                 │ │
│  │  tool.go          invokableToolImpl HTTP 工具调用实现                          │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                             │
│                                       ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ internal/config/ - 配置加载                                                     │ │
│  │                                                                               │ │
│  │  agent.go         Agent 配置结构体、LoadAgents() 加载                          │ │
│  │  agent_tool.go    CollectToolCategories() 递归收集工具类别                      │ │
│  │  const.go         常量定义 (AgentType, ToolCategoryCondition)                  │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────┘
                    │                                      │
                    │ 调用                                 │ 解析事件
                    ▼                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                                    wga-sandbox                                       │
│                              低级 API - 沙箱容器交互                                  │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ api.go                                                                        │ │
│  │  ├─ Run(ctx, opts)        执行任务，返回 <-chan string                        │ │
│  │  └─ Cleanup(ctx, runID)   清理沙箱环境                                         │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                             │
│                                       ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ Run() 执行流程                                                                │ │
│  │                                                                               │ │
│  │  ┌────────────────┐   ┌────────────────┐   ┌────────────────┐               │ │
│  │  │ manager.Create │──▶│ runner.BeforeRun│──▶│ runner.Run    │               │ │
│  │  │   创建沙箱      │   │   准备环境      │   │   执行任务    │               │ │
│  │  └────────────────┘   └────────────────┘   └───────┬────────┘               │ │
│  │                                                     │                        │ │
│  │  ┌────────────────┐   ┌────────────────┐           │                        │ │
│  │  │ manager.Cleanup│◀──│ runner.AfterRun│◀──────────┘                        │ │
│  │  │   清理沙箱      │   │   复制输出      │                                    │ │
│  │  └────────────────┘   └────────────────┘                                    │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                             │
│                                       ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ internal/sandbox/ - 沙箱管理                                                   │ │
│  │                                                                               │ │
│  │  Manager                                                                       │ │
│  │   ├─ Create(ctx, runID, cfg)  创建沙箱实例                                    │ │
│  │   ├─ Get(runID)               获取沙箱实例                                     │ │
│  │   └─ Cleanup(ctx, runID)      清理沙箱                                        │ │
│  │                                                                               │ │
│  │  Sandbox 接口                                                                  │ │
│  │   ├─ Prepare(ctx)                  创建工作目录                               │ │
│  │   ├─ Execute(ctx, args)            异步执行命令                               │ │
│  │   ├─ ExecuteSync(ctx, args)        同步执行命令                               │ │
│  │   ├─ CopyToSandbox(ctx, local, rel)复制文件到沙箱                             │ │
│  │   ├─ CopyFromSandbox(ctx, local)   复制文件到本地                             │ │
│  │   └─ Cleanup(ctx)                  清理工作目录                               │ │
│  │                                                                               │ │
│  │  实现模式                                                                      │ │
│  │   ├─ reuseSandbox   复用已启动容器     ✅ 已实现                               │ │
│  │   └─ oneshotSandbox 每次启动新容器     ✅ 已实现                               │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                             │
│                                       ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ internal/runner/ - 运行器                                                      │ │
│  │                                                                               │ │
│  │  Runner 接口                                                                   │ │
│  │   ├─ BeforeRun(ctx)  准备环境（创建配置、复制文件）                            │ │
│  │   ├─ Run(ctx)        执行任务，返回 JSON 事件流                               │ │
│  │   └─ AfterRun(ctx)   后处理（复制输出）                                        │ │
│  │                                                                               │ │
│  │  实现类型                                                                      │ │
│  │   └─ opencode.Runner   opencode 智能体（通过 HTTP API + SSE）                 │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                             │
│                                       ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ wga-sandbox-converter/ - Eino 事件转换器                                        │ │
│  │                                                                               │ │
│  │  EinoConverter 接口                                                           │ │
│  │   └─ Convert(line string) → []*schema.Message                                 │ │
│  │                                                                               │ │
│  │  ConvertToEinoIterator(ctx, runnerType, outputCh) → *AsyncIterator           │ │
│  │                                                                               │ │
│  │  opencodeConverter 实现                                                        │ │
│  │   ├─ text      → Message{Content}                                            │ │
│  │   ├─ reasoning → Message{ReasoningContent}                                   │ │
│  │   ├─ tool_use  → Message{ToolCalls}                                          │ │
│  │   ├─ file      → Message{Content: "[file] ..."}                              │ │
│  │   └─ error     → Message{Content: "[error] ..."}                             │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        │ JSON 字符串流
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                                     ag-ui-util                                       │
│                              协议转换层 - AG-UI 事件生成                              │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ Translator 接口                                                                │ │
│  │  ├─ Translate(ctx, line) → []Event    转换单个事件                             │ │
│  │  ├─ Finish() → []Event                生成结束事件                             │ │
│  │  └─ TranslateStream(ctx, in) → chan   转换事件流                               │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                     │
│  ┌──────────────────────────────┐    ┌────────────────────────────────────────┐   │
│  │ OpencodeTranslator           │    │ EinoTranslator                         │   │
│  │                              │    │                                        │   │
│  │ 输入: opencode JSON 字符串    │    │ 输入: eino AgentEvent                  │   │
│  │                              │    │                                        │   │
│  │ {"type":"text",              │    │ AgentEvent{                            │   │
│  │  "part":{"text":"..."}}      │    │   Output: {                            │   │
│  │                              │    │     MessageOutput: {                   │   │
│  │ 输出: AG-UI Events           │    │       Message: {                       │   │
│  │  ├─ RunStarted               │    │         Content: "...",                │   │
│  │  ├─ TextMessageStart         │    │         ToolCalls: [...]               │   │
│  │  ├─ TextMessageContent       │    │       }                                │   │
│  │  ├─ TextMessageEnd           │    │     }                                  │   │
│  │  ├─ ToolCallStart            │    │   }                                    │   │
│  │  ├─ ToolCallArgs             │    │                                        │   │
│  │  ├─ ToolCallEnd              │    │ 输出: AG-UI Events (同左)              │   │
│  │  ├─ ToolCallResult           │    │                                        │   │
│  │  └─ RunFinished              │    │                                        │   │
│  │                              │    │                                        │   │
│  │ 使用场景:                    │    │ 使用场景:                               │   │
│  │ wga-sandbox 直接输出转换      │    │ wga.Run() 返回的事件转换                │   │
│  └──────────────────────────────┘    └────────────────────────────────────────┘   │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐ │
│  │ 辅助组件                                                                      │ │
│  │  ├─ MessageState      管理 TEXT_MESSAGE/REASONING 状态机                      │ │
│  │  ├─ BaseState         基础状态（RunStarted/RunFinished）                       │ │
│  │  ├─ AgentActivitySimple  单智能体活动状态                                     │ │
│  │  └─ EventsToJSONChannel  事件流 → JSON 字符串流                               │ │
│  └───────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

## 数据流

```
路径 A: wga 高级 API
───────────────────

  用户请求
       │
       ▼
  ┌─────────────┐
  │  wga.Run()  │
  └──────┬──────┘
         │
         ▼
  ┌───────────────────┐
  │ factory.NewAgent  │
  │  (type=sandbox)   │
  └────────┬──────────┘
           │
           ▼
   ┌───────────────────────────────────────────────────────────────────────┐
   │ sandboxAgent.Run()                                                    │
   │                                                                       │
   │  ┌────────────────┐   ┌─────────────────┐   ┌────────────────────┐  │
   │  │wga_sandbox.Run │──▶│ opencode runner │──▶│ JSON 字符串流      │  │
   │  └────────────────┘   └─────────────────┘   └──────────┬─────────┘  │
   │                                                        │            │
   │  ┌─────────────────────────────────────────────────────▼──────────┐ │
   │  │ wga_sandbox_converter.ConvertToEinoIterator()                  │ │
   │  │ JSON → AgentEvent                                              │ │
   │  └─────────────────────────────────────────────────────┬──────────┘ │
   │                                                        │            │
   │  ┌─────────────────────────────────────────────────────▼──────────┐ │
   │  │ 返回 *adk.AsyncIterator[*adk.AgentEvent]                        │ │
   │  └────────────────────────────────────────────────────────────────┘ │
   └───────────────────────────────────────────────────────────────────────┘
                                                           │
                                                           ▼
                                               ┌────────────────────────┐
                                               │ *adk.AsyncIterator     │
                                               │ [*adk.AgentEvent]      │
                                               └───────────┬────────────┘
                                                           │
                                                           ▼
  ┌───────────────────────────────────────────────────────────────────────────┐
  │ EinoTranslator.TranslateStream(ctx, iter)                                 │
  │                                                                           │
  │  AgentEvent → AG-UI Event                                                 │
  │   ├─ Message.Content         → TextMessageContent                         │
  │   ├─ Message.ReasoningContent→ TextMessageContent (引用格式)               │
  │   └─ Message.ToolCalls       → ToolCallStart/Args/End/Result              │
  └───────────────────────────────────────────────────────────────────────────┘
                                                           │
                                                           ▼
                                               ┌────────────────────────┐
                                               │ AG-UI Events (JSON)    │
                                               └───────────┬────────────┘
                                                           │
                                                           ▼
                                               ┌────────────────────────┐
                                               │ SSE 响应给前端         │
                                               └────────────────────────┘


路径 B: wga-sandbox 低级 API
────────────────────────────

  用户请求
       │
       ▼
  ┌────────────────────────┐
  │ wga_sandbox.Run()      │
  └───────────┬────────────┘
              │
              ▼
  ┌───────────────────────────────────────────────────────────────────────┐
  │ manager.Create(ctx, runID, cfg)                                       │
  │  ├─ 创建沙箱工作目录                                                   │
  │  └─ 返回 Sandbox 实例                                                  │
  └───────────────────────────────────────────────────────────────────────┘
              │
              ▼
  ┌───────────────────────────────────────────────────────────────────────┐
  │ runner.BeforeRun(ctx)                                                 │
  │  └─ opencode: 创建 opencode.json, 复制 skills/tools/input             │
  └───────────────────────────────────────────────────────────────────────┘
              │
              ▼
  ┌───────────────────────────────────────────────────────────────────────┐
  │ runner.Run(ctx) → chan string                                         │
  │                                                                       │
  │  opencode: 通过 HTTP API 调用 opencode，接收 SSE 事件流                 │
  │                                                                       │
  │  输出流:                                                               │
  │   {"type":"text","part":{"text":"..."}}                               │
  │   {"type":"tool_use","part":{"tool":"bash",...}}                      │
  │   {"type":"reasoning","part":{"text":"..."}}                          │
  └───────────────────────────────────────────────────────────────────────┘
              │
              ▼
  ┌───────────────────────────────────────────────────────────────────────┐
  │ runner.AfterRun(ctx)                                                  │
  │  └─ 复制沙箱输出到 OutputDir                                           │
  └───────────────────────────────────────────────────────────────────────┘
              │
              ▼
  ┌───────────────────────────────────────────────────────────────────────┐
  │ manager.Cleanup(ctx, runID) (除非 SkipCleanup=true)                   │
  │  └─ 删除沙箱工作目录                                                   │
  └───────────────────────────────────────────────────────────────────────┘
              │
              ▼
  ┌───────────────────────────────────────────────────────────────────────┐
  │ OpencodeTranslator.TranslateStream(ctx, outputCh)                     │
  │                                                                       │
  │  JSON line → AG-UI Event                                              │
  │   ├─ text      → TextMessageContent                                   │
  │   ├─ reasoning → TextMessageContent (引用格式)                         │
  │   ├─ tool_use  → ToolCallStart/Args/End/Result                        │
  │   └─ error     → TextMessageContent ("[error] ...")                   │
  └───────────────────────────────────────────────────────────────────────┘
              │
              ▼
  ┌────────────────────────┐
  │ AG-UI Events (JSON)    │
  └───────────┬────────────┘
              │
              ▼
  ┌────────────────────────┐
  │ SSE 响应给前端         │
  └────────────────────────┘
```

## 依赖关系

```
       ┌─────────────────┐
       │    应用层        │
       └────────┬────────┘
                │
         ┌──────┴──────┐
         │             │
         ▼             ▼
    ┌────────┐    ┌─────────────┐
    │  wga   │    │ wga-sandbox │
    └───┬────┘    └──────┬──────┘
        │                │
        │    ┌───────────┤
        │    │           │
        │    ▼           ▼
        │ ┌────────────────────────┐
        │ │ wga-sandbox-converter  │
        │ └────────────────────────┘
        │                │
        └───────┬────────┘
                │
                ▼
    ┌────────────┐
    │ ag-ui-util │
    └────────────┘

    依赖说明:
     ├─ wga 依赖 wga-sandbox (sandbox agent 内部调用)
     ├─ wga-sandbox 提供 wga-sandbox-converter (JSON → eino AgentEvent)
     ├─ wga 不依赖 ag-ui-util (返回 eino AgentEvent，由应用层选择是否转换)
     ├─ wga-sandbox 不依赖 ag-ui-util (输出 JSON 字符串流，由调用方选择是否转换)
     └─ ag-ui-util 完全独立 (可单独使用)

    使用场景:
     ├─ examples/agui-demo/backend-wga: wga.Run() → EinoTranslator → AG-UI
     └─ examples/agui-demo/backend-wga-sandbox: wga_sandbox.Run() → OpencodeTranslator → AG-UI
```

## 配置映射

```
用户代码                           wga 内部                      wga-sandbox 内部
───────                           ─────────                    ────────────────

wga.WithModelConfig(      ─────▶  options.Model         ─────▶  ModelConfig
  Provider: "zhipu",
  BaseURL: "https://...",
  APIKey: "sk-xxx",
  Model: "glm-5",
  ModelName: "GLM-5",
  Params: {...}
)

wga.WithToolConfig(       ─────▶  options.Tools[]       ─────▶  配置文件工具的认证
  Title: "天气查询",                                          (通过 Title 匹配)
  APIAuth: {...}
)

wga.WithExtraTool(        ─────▶  options.ExtraTools[]  ─────▶  额外工具（运行时传入）
  OpenAPI3Schema: doc,                                        .OpenAPI3Schema
  APIAuth: {...}                                              .APIAuth
)

wga.WithMCP(               ─────▶  options.MCPs[]        ─────▶  MCP 服务器
  Name: "Jira工单",                                            .Name
  SSEURL: "https://...",                                       .SSEURL
)

wga.WithInputDir("...")    ─────▶  options.Workspace.InputDir   ─────▶  InputDir
wga.WithOutputDir("...")   ─────▶  options.Workspace.OutputDir  ─────▶  OutputDir

wga.WithRunSession(       ─────▶  options.RunSession    ─────▶  RunSession
  ThreadID: "thread-123",                                       .ThreadID
  RunID: "run-456"                                              .RunID
)

wga.WithMessages(         ─────▶  options.Messages      ─────▶  Messages
  {Role: "user", Content: "..."}                                .Role
)                                                               .Content

配置文件 (YAML)                     internal/config              runner 内部
─────────────                       ─────────────              ────────────

agent.yaml  ──────────────────▶   config.Agent
  id: code-agent                      .ID
  type: sandbox                       .Type
  name: 代码助手                       .Name
  description: 代码生成和修改          .Description
  prompt_relative_path: ./prompt.md  ─────────▶  .Prompt        ───────▶   Instruction

  configure:
    max_iterations: 10               .Configure.MaxIterations
    enable_thinking: true  ─────────▶ .Configure.EnableThinking ──────▶  EnableThinking
    sandbox:
      type: reuse                    .Configure.Sandbox.Type
      host: localhost                .Configure.Sandbox.Host     ──────▶  SandboxConfig

  tool_categories:
    - category: search               .ToolCategories
      condition: optional
      tools:
        - path: configs/tools/search.json   .Tools[].SchemaPath (相对程序运行目录)
          auth_required: true               .Tools[].AuthRequired
          operations:                       .Tools[].Operations
            - operation_id: web_search

  skills:
    - dir: configs/skills/coding     .Skills[].Dir (相对程序运行目录)

  sub_agents:
    - relative_path: ./sub-agent/config.yaml  (相对当前配置文件目录)
```

## 配置路径规则

| 配置项 | 字段名 | 相对路径基准 |
|--------|--------|-------------|
| `sub_agents` | `relative_path` | 当前配置文件目录 |
| `prompt_relative_path` | `relative_path` | 当前配置文件目录 |
| `tools` | `path` | 程序运行目录 |
| `skills` | `dir` | 程序运行目录 |

## 智能体类型

| 类型 | 说明 |
|------|------|
| react | ReAct 模式原子智能体，使用 eino adk 实现 |
| sandbox | 沙箱执行智能体，在隔离容器中运行 opencode |
| sequential | 顺序执行多个子智能体 |
| loop | 循环执行子智能体 |
| parallel | 并行执行多个子智能体 |
| deep | 深度思考智能体，递归分解任务 |
| supervisor | 监督者模式，由主智能体协调子智能体 |

## 使用

```go
ctx := context.Background()

// 初始化
wga.Init(ctx, "/path/to/config.yaml")

// 检查运行条件
result, _ := wga.CheckOptions(ctx, "agent-id",
    wga.WithModelConfig(wga_option.ModelConfig{
        Model:   "glm-5",
        BaseURL: "https://api.example.com/v1",
        APIKey:  "sk-xxx",
    }),
    wga.WithToolConfig(wga_option.ToolConfig{
        Title:   "天气查询",
        APIAuth: &util.ApiAuthWebRequest{Type: "header", Name: "Authorization", Value: "Bearer xxx"},
    }),
    wga.WithExtraTool(wga_option.ExtraTool{
        OpenAPI3Schema: openAPIDoc,
        APIAuth:        &util.ApiAuthWebRequest{Type: "header", Name: "X-API-Key", Value: "xxx"},
    }),
)

// 执行
runSession, iter, _ := wga.Run(ctx, "agent-id",
    wga.WithModelConfig(wga_option.ModelConfig{
        Model:   "glm-5",
        BaseURL: "https://api.example.com/v1",
        APIKey:  "sk-xxx",
    }),
    wga.WithMessages([]adk.Message{
        &schema.Message{Role: schema.User, Content: "任务描述"},
    }),
    wga.WithMCP(wga_option.MCP{
        Name:   "Jira工单",
        SSEURL: "https://jira.example.com/sse",
    }),
)

// 获取结果
for {
    event, ok := iter.Next()
    if !ok {
        break
    }
    if event.Output != nil && event.Output.MessageOutput != nil {
        fmt.Println(event.Output.MessageOutput.Message.Content)
    }
}
```

## API

| 函数 | 说明 |
|------|------|
| `Init(ctx, configPath)` | 初始化配置 |
| `GetAgentToolCategories(id)` | 获取智能体及其子智能体的工具类别配置 |
| `CheckOptions(ctx, id, opts...)` | 检查运行条件（模型配置、工具条件、额外工具冲突） |
| `Run(ctx, id, opts...)` | 执行智能体 |
| `Cleanup(ctx, runID)` | 清理资源 |

## 选项

| 选项 | 说明 |
|------|------|
| `WithModelConfig` | 模型配置（必须） |
| `WithToolConfig` | 工具配置（配置文件工具的认证信息） |
| `WithExtraTool` | 额外工具（运行时动态添加，非配置文件中定义的工具） |
| `WithMCP` | MCP 服务器（运行时动态添加） |
| `WithInputDir` | 输入目录 |
| `WithOutputDir` | 输出目录 |
| `WithRunSession` | 会话标识 |
| `WithMessages` | 消息列表（历史消息 + 当前问题，最后一条必须是 User 消息） |

## MCP 服务器

`WithMCP` 用于添加 MCP (Model Context Protocol) 服务器，允许智能体通过 SSE 协议与外部工具交互。

```go
wga.WithMCP(wga_option.MCP{
    Name:   "Jira工单",
    SSEURL: "https://jira.example.com/sse",
}),
wga.WithMCP(wga_option.MCP{
    Name:   "Confluence",
    SSEURL: "https://confluence.example.com/sse",
}),
```

MCP 配置会被传递到 opencode runner，生成 opencode.json 中的 mcp 配置：

```json
{
  "mcp": {
    "Jira工单": {
      "type": "remote",
      "url": "https://jira.example.com/sse",
      "enabled": true
    },
    "Confluence": {
      "type": "remote",
      "url": "https://confluence.example.com/sse",
      "enabled": true
    }
  }
}
```

## 工具配置

### ToolConfig vs ExtraTool

| 特性 | ToolConfig | ExtraTool |
|------|------------|-----------|
| 用途 | 配置文件工具的认证信息 | 运行时添加额外工具 |
| OpenAPI Schema | 不需要（从配置文件读取） | 必须 |
| 认证方式 | `ApiAuthWebRequest` | `ApiAuthWebRequest` |
| 标题匹配 | 通过 Title 匹配配置文件工具 | 自动从 Schema.Info.Title 读取 |
| 冲突检查 | 同 Title 不允许重复 | 不允许与配置文件工具或已有额外工具重名 |

### 工具类别条件

| 条件 | 说明 |
|------|------|
| `none` | 无需检查，该类别下的工具都是可选项 |
| `optional` | 该类别下至少有一个工具完成配置 |
| `required` | 该类别下所有工具完成配置 |

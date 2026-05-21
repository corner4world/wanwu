# mcp2skill

连接 MCP Server 获取工具列表，转换为 Skill 格式的结构化 Markdown 文档。

## Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `streamableUrl` | string | Conditional | MCP Server Streamable HTTP 端点 URL（与 sseUrl 至少填一个） |
| `sseUrl` | string | Conditional | MCP Server SSE 端点 URL（与 streamableUrl 至少填一个） |
| `name` | string | No | Skill 名称，留空时从工具列表自动推断 |
| `description` | string | No | Skill 描述，留空时自动生成 |
| `transport` | string | No | 传输类型：`streamable`（默认）或 `sse` |
| `output` | string | No | 输出目录，默认当前目录 |
| `timeout` | string | No | 连接超时时间，默认 30s |

## Example

### 通过 streamable URL 转换

```bash
mcp2skill name=天气查询 streamableUrl=http://192.168.0.21:8081/mcp/server/streamable?key=xxx output=./skills
```

### 通过 SSE URL 转换

```bash
mcp2skill name=地图服务 sseUrl=http://192.168.0.21:8081/mcp/server/sse?key=xxx transport=sse output=./skills
```

### 输出结构

转换成功后，在输出目录下生成：

```
{output}/{skillName}/
├── SKILL.md                        # Skill 入口概览
└── references/
    └── operations/                 # 每个工具一个详情文件
        └── {tool-name}.md
```

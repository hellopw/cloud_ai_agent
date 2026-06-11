# Cloud AI Agent — MCP Server 设计方案

## 1. 背景与目标

当前 `cloud_ai_agent` Go 后端通过 REST API（`/api/...`）管理 Agent、Template、Prompt、Skill、Tool、ProviderConfig、AgentTeam 等资源。Agent 运行时容器内已具备 MCP **客户端**能力（`mcp-client.js`），可调用外部 MCP 工具。

**目标：** 在 Go 后端增加 MCP **服务端**，将平台管理能力暴露为 MCP Tools。外部 AI Agent（如 Codex）可通过 SSE 传输连接该 MCP Server，调用 `create_agent`、`build_agent`、`start_agent` 等工具，实现 **自举（bootstrapping）**——Agent 能够通过 MCP 协议创建和管理新的 Agent。

## 2. 技术方案

### 2.1 传输协议

采用 **SSE（Server-Sent Events）传输**，在现有 HTTP 服务器上挂载 `POST /api/mcp` 端点。MCP 协议为标准 JSON-RPC 2.0 over SSE。

### 2.2 MCP 库选型

使用 `github.com/mark3labs/mcp-go`，该库提供：
- 完整的 MCP SSE Server 实现
- 工具注册与 JSON Schema 自动生成
- Session 管理

### 2.3 Go 版本要求

`mark3labs/mcp-go` 需要 **Go 1.21+**。当前 `go.mod` 为 `go 1.18`，需将版本升级至 `go 1.21`（或更高）。

## 3. 模块结构

```
backend/internal/mcp/
├── server.go              # MCP Server 初始化与路由注册
├── tools_agents.go         # Agent CRUD + 生命周期工具
├── tools_instances.go      # Instance 工具
├── tools_templates.go      # Template 工具
├── tools_prompts.go        # Prompt 工具
├── tools_skills.go         # Skill 工具
├── tools_tools.go          # Tool (自定义工具) 工具
├── tools_agentteams.go     # AgentTeam CRUD + 生命周期工具
├── tools_provider_configs.go # ProviderConfig 工具
├── tools_resources.go      # Resource 工具
└── tools_health.go         # Health 检查工具
```

每个 `tools_*.go` 文件负责注册一类实体的 MCP Tools，所有 handler 共享注入的 `*store.Store` 和 `*service.AgentService`。

## 4. MCP Tools 清单

### 4.1 Agent

| Tool | 参数 | 说明 |
|------|------|------|
| `list_agents` | 无 | 列出所有 Agent |
| `get_agent` | `id: string` | 获取单个 Agent |
| `create_agent` | `name`, `template_id`, `repo_url`, `branch`, `git_username`, `git_password` | 创建 Agent |
| `update_agent` | `id` + 同 create | 更新 Agent（仅 draft/failed 状态） |
| `delete_agent` | `id: string` | 删除 Agent |
| `build_agent` | `id: string` | 异步构建 Docker 镜像，返回 "build started" |
| `get_build_log` | `id: string` | 获取构建日志 |

### 4.2 Instance

| Tool | 参数 | 说明 |
|------|------|------|
| `list_instances` | 无 | 列出所有实例 |
| `get_instance` | `id: string` | 获取单个实例 |
| `start_instance` | `agent_id: string` | 启动 Agent 容器实例 |
| `delete_instance` | `id: string` | 删除实例 |

### 4.3 Template

| Tool | 参数 | 说明 |
|------|------|------|
| `list_templates` | 无 | 列出模板 |
| `get_template` | `id: string` | 获取模板详情（含关联的 prompts/skills/tools） |
| `create_template` | `name`, `description`, `dockerfile_content` | 创建模板 |
| `update_template` | `id` + 同 create | 更新模板 |
| `delete_template` | `id: string` | 删除模板 |
| `update_template_bindings` | `id`, `prompt_ids`, `skill_ids`, `tool_ids` | 更新模板绑定关系 |

### 4.4 Prompt / Skill / Tool / Resource / ProviderConfig

每类实体提供标准 CRUD 工具：`list_*`, `get_*`, `create_*`, `update_*`, `delete_*`。

注意事项：
- `ProviderConfig` 的 `api_key` 字段在输出中脱敏（不返回），仅作为写入参数
- `Agent` 的 `git_password` 同理

### 4.5 AgentTeam

| Tool | 参数 | 说明 |
|------|------|------|
| `list_agent_teams` | 无 | 列出团队 |
| `get_agent_team` | `id: string` | 获取团队详情（含 members） |
| `create_agent_team` | 完整团队配置 | 创建团队 |
| `update_agent_team` | `id` + 配置 | 更新团队 |
| `delete_agent_team` | `id: string` | 删除团队 |
| `build_agent_team` | `id: string` | 异步构建团队镜像 |
| `get_team_build_log` | `id: string` | 获取团队构建日志 |
| `start_team_instance` | `team_id: string` | 启动团队实例 |

### 4.6 Health

| Tool | 参数 | 说明 |
|------|------|------|
| `health` | 无 | 返回服务健康状态 |

## 5. 集成入口

### 5.1 `cmd/server/main.go` 变更

```go
import "cloud_ai_agent/internal/mcp"

// 在现有初始化代码之后添加：
mcpServer := mcp.NewServer(s, agentSvc)
mcpHandler := mcpServer.Handler()
```

### 5.2 `internal/api/router.go` 变更

```go
// 在 NewRouter 中添加（与其他 API 路由并行）：
mux.HandleFunc("/api/mcp", func(w http.ResponseWriter, r *http.Request) {
    if r.Method == "POST" {
        mcpHandler.ServeHTTP(w, r)
        return
    }
    methodNotAllowed(w)
})
```

MCP 端点复用现有 CORS 中间件。

## 6. 安全考虑

- MCP 端点当前与 REST API 同级，无额外认证
- 敏感字段（`api_key`, `git_password`）仅作为输入参数，不出现在返回 schema 中
- 后续可按需添加 API Key 或 Token 认证

## 7. 测试计划

1. **单元测试**：每个 tool handler 使用内存 SQLite 测试 store 进行独立验证
2. **集成测试**：启动完整服务，使用 MCP 客户端执行完整流程：
   ```
   create_template → create_agent → build_agent → get_build_log
   → start_instance → get_instance → delete_instance → delete_agent
   ```
3. **端到端验证**：用 Codex 连接该 MCP Server，验证自举能力

## 8. 依赖变更

| 操作 | 包 | 原因 |
|------|-----|------|
| 新增 | `github.com/mark3labs/mcp-go` | MCP 协议实现 |
| 升级 | `go 1.18` → `go 1.21` | mcp-go 最低版本要求 |


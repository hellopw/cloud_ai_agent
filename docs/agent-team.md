# Agent Team — 多 Agent 团队协作

## 概念

AgentTeam 将多个 Agent 打包在**同一个容器**中运行，实现**主从协作模式**（Master-Worker）。一个 Leader Agent 负责任务理解和分派，多个 Worker Agent 各司其职执行具体任务。每个 Agent 拥有**独立的上下文**、工具集和 LLM 配置，通过文件系统共享工作区。

## 核心特性

### 1. 主从协作

用户的消息始终发送给 Leader。Leader 通过内置的 `delegate_task` 工具将子任务分派给 Worker：

```
User -> Leader (大脑+分派) -> Worker A (代码审查)
                           -> Worker B (测试生成)
                           -> Worker C (文档编写)
```

Worker 的执行结果返回给 Leader，Leader 汇总后回复用户。前端 Chat 窗口始终保持单一对话入口。

### 2. 独立上下文

每个成员 Agent 拥有独立的：
- **LLM 配置**（Provider + Model）：成员可使用不同的 Provider（如 Leader 用 OpenAI，Worker 用 Claude）
- **系统提示词**（可从 Template 继承，也可单独覆盖 `system_prompt_override`）
- **Skills**（成员级 + Team 级合并，同名文件成员级覆盖 Team 级）
- **Prompts**（同上合并规则）
- **Tools**（Template 绑定的 tools + 成员额外绑定的 tools）

### 3. 共享工作区

所有 Agent 共享同一个 `/workspace`（Git 仓库的 bind mount），支持文件级协作。Worker 可以修改文件，Leader 通过 `bash` / `read_file` 等工具检查结果。

## 数据模型

### agent_teams 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT PK | UUID |
| name | TEXT | 团队名称 |
| template_id | TEXT FK | Dockerfile 模板 |
| repo_url | TEXT | Git 仓库地址 |
| branch | TEXT | 分支 |
| git_username | TEXT | Git 认证用户名 |
| git_password | TEXT | Git 认证密码/token |
| image_tag | TEXT | Docker 镜像标签 |
| status | TEXT | draft/building/ready/failed |
| error_msg | TEXT | 构建失败原因 |

### team_members 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT PK | UUID |
| team_id | TEXT FK | 所属团队 |
| name | TEXT | 成员名称（如 "code-reviewer"） |
| role | TEXT | leader 或 worker |
| agent_template_id | TEXT FK | 成员使用的 Template |
| provider_config_id | TEXT FK | LLM 配置 |
| system_prompt_override | TEXT | 覆盖默认系统提示词 |
| sequence | INTEGER | 排序 |

### 绑定关系表

- **team_member_prompts/skills/tools**: 成员额外绑定的 prompts/skills/tools
- **agent_team_prompts/skills**: Team 级 prompts/skills（所有成员共享）

### Instance 新增字段

| 字段 | 类型 | 说明 |
|------|------|------|
| team_id | TEXT | 归属的 AgentTeam ID（空字符串=普通 Agent 实例） |

## API 路由

| 路由 | 方法 | 说明 |
|------|------|------|
| `/api/agent-teams` | GET | 列出所有团队 |
| `/api/agent-teams` | POST | 创建团队 |
| `/api/agent-teams/{id}` | GET | 获取团队详情（含 members） |
| `/api/agent-teams/{id}` | PUT | 更新团队（仅 draft/failed） |
| `/api/agent-teams/{id}` | DELETE | 删除团队 |
| `/api/agent-teams/{id}/build` | POST | 触发构建（异步，返回 202） |
| `/api/agent-teams/{id}/log` | GET | 查看构建日志 |
| `/api/agent-teams/{id}/start` | POST | 启动团队实例 |

### AgentTeam 对象结构

```json
{
  "id": "uuid",
  "name": "my-team",
  "template_id": "uuid",
  "repo_url": "https://github.com/user/repo.git",
  "branch": "main",
  "git_username": "",
  "git_password": "",
  "image_tag": "cloud-agent-team/uuid:latest",
  "status": "ready",
  "error_msg": "",
  "prompt_ids": ["uuid1"],
  "skill_ids": ["uuid2"],
  "members": [
    {
      "id": "uuid",
      "team_id": "uuid",
      "name": "leader",
      "role": "leader",
      "agent_template_id": "uuid",
      "provider_config_id": "uuid",
      "prompt_ids": [],
      "skill_ids": [],
      "tool_ids": [],
      "system_prompt_override": "",
      "sequence": 0
    },
    {
      "id": "uuid",
      "name": "code-reviewer",
      "role": "worker",
      "agent_template_id": "uuid",
      "provider_config_id": "uuid",
      "system_prompt_override": "You are a code reviewer...",
      "sequence": 1
    }
  ]
}
```

## 构建流程

POST /api/agent-teams/{id}/build 触发，异步后台执行：

1. 状态 -> `building`，清空并重建 `builds/{team_id}/`
2. 创建 `builds/{team_id}/build.log`
3. Git clone 代码仓库到 `builds/{team_id}/repo/`
4. 生成 Team 版 Dockerfile（使用 `team_dockerfile.go` 模板）
5. 复制 `team-server.js` 和 `mcp-client.js` 到构建上下文
6. 为每个成员创建子目录 `builds/{team_id}/agents/{name}/`：
   - 写入 tool extensions（Template 绑定的 + 成员额外绑定的）
   - 合并 prompts/skills（Team 级 + 成员级，同名文件成员级覆盖）
   - 从 ProviderConfig 读取 LLM 配置
7. 写入 Team 级 prompts/skills 到 `team-prompts/` 和 `team-skills/`
8. 写入 `team-manifest.json`（包含所有成员的 LLM 配置）
9. `docker build -t cloud-agent-team/{team_id}:latest`
10. 状态 -> `ready` 或 `failed`

### team-manifest.json 格式

```json
{
  "team_name": "my-team",
  "members": [
    {
      "name": "leader",
      "role": "leader",
      "provider": "openai-codex",
      "model_id": "gpt-5.1-codex-max",
      "api_key": "sk-...",
      "base_url": "",
      "system_prompt_override": ""
    },
    {
      "name": "code-reviewer",
      "role": "worker",
      "provider": "anthropic",
      "model_id": "claude-sonnet-4-6",
      "api_key": "sk-ant-...",
      "base_url": "",
      "system_prompt_override": "You are a code reviewer..."
    }
  ],
  "team_prompts_dir": "/app/team-prompts",
  "team_skills_dir": "/app/team-skills"
}
```

## 容器内 Wrapper（team-server.js）

与单 Agent 的 `server.js` 不同，`team-server.js` 启动时读取 `team-manifest.json`，为每个成员创建独立的 `Agent` 实例。

### 初始化流程

1. 读取 `/app/team-manifest.json`
2. 遍历 members，为每个 member 调用 `setupAgent(memberConfig)` 创建 Agent 实例
3. Worker Agent 预初始化（eager init），Leader 延迟初始化（lazy init）
4. Leader 额外注册 `delegate_task` 工具

### HTTP 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/status` | GET | 返回所有成员名称、角色和就绪状态 |
| `/health` | GET | 健康检查 `{status: "ok"}` |
| `/chat` | POST | SSE 流，始终路由到 Leader |
| `/abort` | POST | 中断 Leader 当前操作 |
| `/internal/agents/:name/chat` | POST | 内部接口：执行指定 Worker |
| `/internal/agents/:name/status` | GET | 内部接口：查询 Worker 状态 |

### delegate_task 工具

Leader 专属工具，参数：
- `worker` (string): 目标 Worker 名称
- `task` (string): 任务描述

内部通过 `delegateToWorker()` 函数调用 Worker Agent，同步等待完成并返回结果。

## 前端页面

| 路由 | 组件 | 功能 |
|------|------|------|
| `/agent-teams` | AgentTeamsPage | 团队列表、构建、启动、查看日志、删除 |
| `/agent-teams/new` | AgentTeamEditPage | 创建团队 |
| `/agent-teams/:id` | AgentTeamEditPage | 编辑团队（成员、绑定、Team 级配置） |

### AgentTeamsPage

团队列表页，与 AgentsPage 风格一致：
- 显示团队名称、状态标签（draft/building/ready/failed）
- 显示成员数量和角色概要
- **Build** 按钮（draft/failed 状态）
- **Start** 按钮（ready 状态）
- **View Log** 按钮（building/failed 状态）
- **Edit** 链接（跳转编辑页）
- **Delete** 按钮

### AgentTeamEditPage

团队编辑页，三个区域：
- **Team Info**: 名称、Template、Git 仓库配置
- **Team-level Prompts & Skills**: 所有成员共享的提示词和技能
- **Members**: 成员列表，每个成员可配置：
  - Name / Role (leader|worker) / Template / Provider Config
  - System Prompt Override
  - 额外 Prompts / Skills / Tools（展开详情编辑）

### InstancesPage 更新

- 团队实例显示 **Team: {name}** 标签
- 新增 **Start New Instance (Team)** 区域，列出所有 ready 状态的团队

## 设计决策

| 决策 | 原因 |
|------|------|
| 所有消息路由到 Leader | 单一聊天入口，简化 UX。Worker 仅通过 delegate_task 调用 |
| delegate_task 同步执行 | 简化 SSE 代理实现，Leader 的流输出完整包含 Worker 结果 |
| Provider 配置嵌入 manifest | 避免环境变量过多，构建时写入 manifest.json |
| Team 级 prompts/skills 优先 | 文件写入顺序（Team 级先、成员级后），同名覆盖 |
| 单容器单端口 | 所有 Agent 在同一个 `localhost:3000` 内，内部通信不暴露端口 |
| 修改 Provider 需重建 | 配置嵌入 manifest，修改后需重新 Build |

## 文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `backend/internal/model/models.go` | 修改 | 新增 AgentTeam, TeamMember, Instance.TeamID |
| `backend/internal/store/store.go` | 修改 | 新增 7 张表 + migration |
| `backend/internal/store/agent_teams.go` | 新增 | AgentTeam CRUD |
| `backend/internal/store/instances.go` | 修改 | 查询增加 team_id |
| `backend/internal/api/handlers.go` | 修改 | 8 个新 handler |
| `backend/internal/api/router.go` | 修改 | `/api/agent-teams` 路由组 |
| `backend/internal/service/agent.go` | 修改 | BuildAgentTeam + StartTeamInstance |
| `backend/internal/codegen/team_dockerfile.go` | 新增 | Team Dockerfile 模板 |
| `container-wrapper/src/team-server.js` | 新增 | 多 Agent 容器 wrapper |
| `frontend/src/api/client.ts` | 修改 | 新增 agentTeamsApi |
| `frontend/src/pages/AgentTeamsPage.tsx` | 新增 | 团队列表页 |
| `frontend/src/pages/AgentTeamEditPage.tsx` | 新增 | 团队编辑页 |
| `frontend/src/App.tsx` | 修改 | 新增导航和路由 |
| `frontend/src/pages/InstancesPage.tsx` | 修改 | 团队实例标识 + 团队启动 |

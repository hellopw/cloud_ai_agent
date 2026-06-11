# Cloud AI Agent — 架构文档

## 概述

Cloud AI Agent 是一个 Agent 管理平台，以 [pi](https://github.com/earendil-works/pi) 为 Agent 运行时，用 **Go** (net/http) 构建管理后端 + **React** (TypeScript, Vite) 构建管理前端，通过 Docker 编排 Agent 容器的完整生命周期。

## 架构图

```
┌─────────────────────────────────────────────────┐
│              React Frontend (SPA)                │
│  Prompts | Skills | Tools | Templates           │
│  Agents | Instances | Models | Resources        │
│  Chat UI (WebSocket <-> SSE)                    │
└──────────────────┬──────────────────────────────┘
                   │ REST + WebSocket
┌──────────────────┴──────────────────────────────┐
│           Go Backend (:8080, 单体)               │
│  ┌──────────┐ ┌──────────┐ ┌────────────────┐  │
│  │ REST API │ │Agent Svc │ │ Proxy (WS<->SSE)│  │
│  │ (net/http│ │(build    │ │ gorilla/       │  │
│  │  手动路由)│ │ start)   │ │ websocket      │  │
│  └──────────┘ └──────────┘ └────────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌────────────────┐  │
│  │ Git Svc  │ │Docker Svc│ │ SQLite         │  │
│  │(exec.Cmd)│ │(exec.Cmd)│ │ (modernc.org/  │  │
│  │          │ │          │ │  sqlite,纯Go)  │  │
│  └──────────┘ └──────────┘ └────────────────┘  │
└──────────────────┬──────────────────────────────┘
                   │ exec.Command("docker")
┌──────────────────┴──────────────────────────────┐
│           Docker Daemon (本地)                   │
│  ┌────────────────────────────────────────────┐ │
│  │ Agent Instance Container                   │ │
│  │  Node.js (Express, pi SDK) (:3000)         │ │
│  │  GET  /status   POST /chat   POST /abort  │ │
│  │  bind mount: /workspace -> 代码仓库         │ │
│  │  内置 tools: bash, read/write/edit_file,  │ │
│  │   search_content, list_files, git          │ │
│  └────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

## 技术栈

| 层 | 技术 | 备注 |
|---|---|---|
| 前端 | React 18 + TypeScript + Vite | 纯 CSS，无 UI 库 |
| 后端 | Go 1.18, net/http | 手动路由，无框架 |
| 数据库 | SQLite (modernc.org/sqlite) | 纯 Go 实现，无需 CGO |
| 容器 | Docker CLI (exec.Command) | 非 Docker SDK |
| Git | Git CLI (exec.Command) | 自动 code.sohuno.com HTTPS->SSH 转换 |
| Agent 运行时 | @earendil-works/pi-agent-core + pi-ai | Node.js，容器内 |
| WebSocket | gorilla/websocket | WS<->SSE 代理 |
| 基础镜像 | private-registry.sohucs.com/domeos-pub/node:22.14.0-alpine3.21 | 含 npm 代理配置 |

## 数据模型 (11 张表)

```sql
prompts          (id, name, description, content, created_at, updated_at)
skills           (id, name, description, content, created_at, updated_at)
tools            (id, name, label, description, dsl_definition, created_at, updated_at)
templates        (id, name, description, dockerfile_content, created_at, updated_at)
template_prompts (template_id -> templates, prompt_id -> prompts, PK)
template_skills  (template_id -> templates, skill_id -> skills, PK)
template_tools   (template_id -> templates, tool_id -> tools, PK)
agents           (id, name, template_id, repo_url, git_username, git_password,
                  branch, image_tag, status, error_msg, resource_ids,
                  created_at, updated_at)
instances        (id, agent_id, container_id, host_port, status, error_msg,
                  created_at, updated_at)
provider_configs (id, name, provider, model_id, api_key, base_url,
                  created_at, updated_at)
resources        (id, name, type, config, created_at, updated_at)
agent_resources  (agent_id -> agents, resource_id -> resources, PK)
```

### 状态机

- **Agent**: `draft` -> `building` -> `ready` | `failed`
- **Instance**: `starting` -> `running` -> `error` | `stopped`

## API 路由

| 前缀 | 方法 | 说明 |
|---|---|---|
| `/api/prompts` | GET/POST | 列表/创建 |
| `/api/prompts/:id` | GET/PUT/DELETE | CRUD |
| `/api/skills` | GET/POST | 列表/创建 |
| `/api/skills/:id` | GET/PUT/DELETE | CRUD |
| `/api/tools` | GET/POST | 列表/创建 |
| `/api/tools/:id` | GET/PUT/DELETE | CRUD |
| `/api/templates` | GET/POST | 列表/创建 |
| `/api/templates/:id` | GET/PUT/DELETE | CRUD，含 dockerfile_content |
| `/api/templates/:id/bind` | PUT | 批量绑定 prompts/skills/tools |
| `/api/agents` | GET/POST | 列表/创建（含 resource_ids） |
| `/api/agents/:id` | GET/PUT/DELETE | CRUD（仅 draft/failed 可编辑） |
| `/api/agents/:id/build` | POST | 触发 Docker 构建（异步） |
| `/api/agents/:id/log` | GET | 查看构建日志 |
| `/api/agents/:id/start` | POST | 启动实例（body: provider_config_id） |
| `/api/instances` | GET | 列表 |
| `/api/instances/:id` | GET/DELETE | 详情/停止删除 |
| `/api/instances/:id/status` | GET | 代理到容器 /status |
| `/api/instances/:id/chat` | WS | WebSocket 代理到容器 SSE |
| `/api/provider-configs` | GET/POST | 模型配置 CRUD |
| `/api/provider-configs/:id` | GET/PUT/DELETE | 支持 openai-codex/anthropic/openai |
| `/api/resources` | GET/POST | 资源 CRUD |
| `/api/resources/:id` | GET/PUT/DELETE | 支持 git/database/knowledge 三类 |
| `/api/health` | GET | 健康检查 |

## 构建流程

POST /api/agents/:id/build 触发，异步后台执行：

1. 状态 -> `building`，清空并重建 `builds/{agent_id}/`
2. 创建 `builds/{agent_id}/build.log`（每次构建独立日志文件）
3. Git clone 代码仓库到 `builds/{agent_id}/repo/`：
   - `code.sohuno.com` HTTPS 地址自动转为 SSH
   - 带用户名/密码时嵌入 URL 凭证
   - 先尝试指定分支 clone，失败则 fallback 到默认分支
4. 代码生成 Dockerfile（模板的 dockerfile_content 或默认模板）
5. 复制 `container-wrapper/src/server.js` 到构建上下文根目录
6. 将绑定的 tools 编译为 TypeScript 扩展写入 `extensions/`
7. 将绑定的 prompts/skills 写入 `pi-prompts/` 和 `pi-skills/`
8. `docker build -t cloud-agent/{agent_id}:latest builds/{agent_id}/`
9. 状态 -> `ready` 或 `failed`（含 error_msg）

## 启动流程

POST /api/agents/:id/start，body 包含 `provider_config_id`：

1. 创建 Instance 记录，状态 `starting`
2. 根据 provider_config_id 查询 api_key 等配置
3. 动态分配端口（3001 + instance_id 哈希值）
4. `docker run -d -p {port}:3000 -v {repo_abs}:/workspace -e AGENT_PROVIDER=... -e AGENT_MODEL=... -e AGENT_API_KEY=... -e AGENT_BASE_URL=... cloud-agent/{id}:latest`
5. 状态 -> `running`

## 容器内 Wrapper

container-wrapper/src/server.js — Express 服务，3000 端口：

| 端点 | 说明 |
|---|---|
| `GET /status` | 返回 `{status, ready, provider, model}` |
| `POST /chat` | SSE 流，body `{message}`，调用 agent.prompt() |
| `POST /abort` | 中止当前 agent 轮次 |

内置 7 个 tools：bash, read_file, write_file, edit_file, search_content, list_files, git。

Provider 通过环境变量指定：AGENT_PROVIDER, AGENT_MODEL, AGENT_API_KEY, AGENT_BASE_URL，支持 openai-codex, anthropic, openai。

## Chat 数据流

```
Browser (WS) -> Go Backend (gorilla/websocket)
  -> HTTP POST /chat -> Container (SSE)
  <- WS: text_delta / tool_call / tool_result / agent_end / error
  <- SSE events from container
```

Go 升级 WebSocket 连接，接收前端 `{type:"chat", message}`，转发到容器 `/chat`，逐行解析 SSE 事件并通过 WS 推回前端。

## 前端页面总览

| 路由 | 组件 | 功能 |
|---|---|---|
| `/prompts` | PromptsPage | 列表、删除 |
| `/prompts/:id` | PromptEditPage | 新建/编辑（name, description, content） |
| `/skills` | SkillsPage | 列表、删除 |
| `/skills/:id` | SkillEditPage | 新建/编辑 |
| `/tools` | ToolsPage | 列表、删除 |
| `/tools/:id` | ToolEditPage | 新建/编辑（name, label, description, DSL JSON） |
| `/templates` | TemplatesPage | 列表、查看 Dockerfile、删除 |
| `/templates/:id` | TemplateEditPage | 新建/编辑、绑定 prompts/skills/tools |
| `/agents` | AgentsPage | 列表、创建、编辑、构建、启动、查看日志/Dockerfile、删除、关联 Resource |
| `/instances` | InstancesPage | 列表、启动（选 Model）、Chat 入口、Stop、查看失败原因 |
| `/instances/:id/chat` | ChatPage | WebSocket 实时对话，流式输出，tool_call 展示 |
| `/models` | ModelsPage | Provider 配置 CRUD（openai-codex/anthropic/openai） |
| `/resources` | ResourcesPage | 资源管理（Git: url/username/token/branch, Database, Knowledge Base） |

## 部署

### 测试环境 (10.19.12.152, binwang)

- 目录：`deploy/`
- 后端二进制：`deploy/cloud-ai-agent`
- 前端静态文件：`deploy/frontend/`（Vite build 产物）
- 容器 wrapper：`deploy/container-wrapper/`
- 启动：`deploy/start.sh`，env: PORT=8080, DB_PATH=data/cloud_ai_agent.db

### 本地开发

```
cd backend && go run ./cmd/server     # :8080
cd frontend && npm run dev            # :5173, proxy /api -> :8080
```

### 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| PORT | 8080 | 后端监听端口 |
| DB_PATH | data/cloud_ai_agent.db | SQLite 路径 |
| FRONTEND_DIR | frontend | 前端静态文件目录 |
| PROJECT_ROOT | 自动检测 | builds/ 和 container-wrapper 的根路径 |
| AGENT_PROVIDER | - | 容器默认 provider |
| AGENT_MODEL | - | 容器默认 model |
| AGENT_API_KEY | - | 容器默认 API key |
| AGENT_BASE_URL | - | 容器默认 base URL |

## 已知问题 & 修复记录

- **React Error #310**: AgentsPage.tsx 中 4 个 useState + 1 个 useEffect 在 `if (loading) return` 之后，hooks 调用顺序不一致 -> 已修复
- **Git clone code.sohuno.com**: HTTPS hang -> 自动转 SSH (git@code.sohuno.com:...)
- **构建独立日志**: 每次构建写入 `builds/{agent_id}/build.log`，可在 Agent 卡片点击 View Log 查看
- **容器 Provider 信息**: 启动 Agent 时选择 Model Config，通过 -e 传入 AGENT_PROVIDER/AGENT_MODEL/AGENT_API_KEY/AGENT_BASE_URL
- **失败实例展示**: Instances 页面同时展示 running 和 error/stopped 状态的实例，并显示 error_msg
- **Agent 编辑限制**: 仅 draft 和 failed 状态的 Agent 可编辑，防止修改已构建成功的 Agent
- **Agent 资源关联**: 创建 Agent 时可 select Git Resource 自动填充仓库信息，也可勾选 database/knowledge 类型资源

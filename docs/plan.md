# 实施计划

## Phase 1 — 基础骨架（预估 3-5 天）

目标：Go 后端 + React 前端项目搭建，prompt/skill/tool 的 CRUD 闭环。

### 1.1 Go 项目初始化
- 目录结构：`backend/cmd/`, `backend/internal/`, `backend/migrations/`
- 依赖：gin 或 chi（路由）、mattn/go-sqlite3（数据库）、golang-migrate（migration）
- 健康检查端点 `GET /api/health`

### 1.2 数据模型 Migration
- 创建 `prompts`, `skills`, `tools` 三张表
- 创建 `templates`, `template_prompts`, `template_skills`, `template_tools` 表
- 创建 `agents`, `instances` 表
- 创建 `chat_sessions`, `chat_messages` 表

### 1.3 Prompts CRUD
- `POST /api/prompts` — 创建
- `GET /api/prompts` — 列表
- `GET /api/prompts/:id` — 详情
- `PUT /api/prompts/:id` — 更新
- `DELETE /api/prompts/:id` — 删除

### 1.4 Skills CRUD
- 同 prompts 的 API 模式，实体为 skills
- API 路径：`/api/skills`

### 1.5 Tools CRUD + 代码生成器
- API 路径：`/api/tools`
- DSL 编辑器前端组件：JSON editor（Monaco Editor）+ 参数表单
- Go 端代码生成器：读取 `dsl_definition` JSON → Go 模板 → 生成 TypeScript 字符串
- `POST /api/tools/:id/preview` — 预览生成的 TypeScript 代码（可选）

### 1.6 React 前端初始化
- Vite + React + TypeScript + React Router
- 页面骨架：侧边栏导航 + 内容区
- Prompts 列表页 + 编辑页
- Skills 列表页 + 编辑页
- Tools 列表页 + 编辑页（含 DSL JSON 编辑器）

## Phase 2 — Agent 生命周期（预估 3-5 天）

目标：模板绑定、Agent 定义、Docker 构建流水线。

### 2.1 Templates CRUD + 绑定 API
- `POST /api/templates` — 创建模板
- `GET /api/templates` — 列表
- `GET /api/templates/:id` — 详情（含绑定的 prompts/skills/tools）
- `PUT /api/templates/:id` — 更新
- `DELETE /api/templates/:id` — 删除
- `PUT /api/templates/:id/bind` — 批量更新绑定关系 `{ prompt_ids, skill_ids, tool_ids }`

### 2.2 Dockerfile 生成器
- Go 模板引擎：根据模板的绑定关系 + 用户自定义片段生成完整 Dockerfile
- 生成的 Dockerfile 内容：
  - `FROM node:22-alpine`
  - `WORKDIR /app`
  - `COPY package.json` → `npm install --ignore-scripts`
  - `COPY pi-skills/ pi-prompts/ extensions/`
  - `COPY server.js`
  - `ENV` 注入（LLM provider keys）
  - `EXPOSE 3000`
  - `CMD ["node", "server.js"]`

### 2.3 Agents CRUD
- `POST /api/agents` — 创建 agent（关联 template_id + repo_url + branch）
- `GET /api/agents` — 列表
- `GET /api/agents/:id` — 详情
- `DELETE /api/agents/:id` — 删除

### 2.4 Git Repo 管理 + Docker Build
- Git 操作：`git clone --depth 1 --branch {branch} {repo_url} {workdir}/{agent_id}/repo`
- Docker build：准备构建上下文（repo + template 文件树 + Dockerfile）→ `docker build`
- `POST /api/agents/:id/build` — 触发构建
- `GET /api/agents/:id/build-status` — 查询构建状态/日志
- 状态流转：`draft` → `building` → `ready` | `failed`

### 2.5 React 模板与 Agent 管理页
- Template 编辑页：基本信息 + 绑定选择器（三列勾选 prompts/skills/tools）
- Agent 列表页 + 创建页（选择模板、填写 repo URL 和分支）
- Agent 详情页：状态显示 + 构建按钮 + 构建日志终端

## Phase 3 — 实例与对话（预估 5-7 天）

目标：容器运行、实时对话、Chat UI。

### 3.1 容器 HTTP Wrapper（Node.js 项目）
- 位置：`container-wrapper/`
- package.json：依赖 `@earendil-works/pi-agent-core`、`express`、`@earendil-works/pi-ai`
- server.js：
  - 启动时加载 skills、prompts、tools
  - `POST /chat`：调用 `agent.prompt()`，EventStream → SSE
  - `POST /tool-result`：注入 tool 结果并 `agent.continue()`
  - `POST /abort`：`agent.abort()`
  - `GET /status`：返回 agent state
- 扩展加载器：动态 import `extensions/` 目录下的 `.ts` 文件

### 3.2 Docker SDK 集成
- Go Docker client：`github.com/docker/docker/client`
- `POST /api/instances` — 启动实例：
  1. 根据 agent_id 获取 agent 和 template 信息
  2. `docker run -d --name {instance_id} -p {动态端口}:3000 -v {repo_path}:/workspace -e OPENAI_API_KEY -e ANTHROPIC_API_KEY {image_tag}`
  3. 记录 container_id 和 host_port
- `DELETE /api/instances/:id` — 停止实例：`docker stop` + `docker rm`
- `GET /api/instances/:id/status` — 查询状态：`docker inspect`

### 3.3 WebSocket ↔ HTTP SSE 代理层
- Go WebSocket endpoint：`/ws/instances/:id/chat`
- WebSocket 消息格式：
  - 前端 → Go：`{ "type": "chat", "message": "hi" }`
  - Go → 前端：`{ "type": "text_delta", "delta": "..." }`
  - Go → 前端：`{ "type": "tool_call", "toolName": "read", "input": {...} }`
  - Go → 前端：`{ "type": "agent_end", "usage": {...} }`
- Go 代理逻辑：
  1. 接收 WebSocket 消息
  2. HTTP POST 到容器 `/chat`，读取 SSE 流
  3. 逐事件转发到 WebSocket
  4. 处理连接断开/超时

### 3.4 React Instances 管理页 + Chat UI
- 实例列表页：启动/停止按钮、状态指示
- Chat 界面：
  - 消息气泡（Markdown 渲染：react-markdown + rehype-highlight）
  - Tool call 卡片（折叠/展开，显示 tool 名、参数、结果）
  - 输入框 + 发送按钮
  - 流式输出：SSE 逐字渲染
  - 会话历史（从后端加载）

## Phase 4 — 完善与测试（预估 2-3 天）

### 4.1 错误处理与日志
- Go 全局 error middleware
- 结构化日志（slog）
- 容器内日志收集
- 前端错误边界 + toast 通知

### 4.2 集成测试
- Go 端：mock Docker client，mock pi HTTP wrapper
- 前端：Vitest + React Testing Library
- 端到端：Playwright（创建 template → 构建 agent → 启动实例 → 对话）

### 4.3 部署配置
- Docker Compose：`go-backend` + `react-frontend`（静态文件或 dev server）
- 环境变量管理：`.env.example`
- 启动脚本：`scripts/dev.sh` / `scripts/dev.ps1`

## 项目目录结构

```
cloud_ai_agent/
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/            # HTTP handlers
│   │   ├── model/          # 数据模型
│   │   ├── service/        # 业务逻辑
│   │   ├── docker/         # Docker SDK 封装
│   │   ├── git/            # Git 仓库操作
│   │   ├── codegen/        # Tool DSL → TypeScript 生成器
│   │   ├── proxy/          # WS ↔ SSE 代理
│   │   └── store/          # SQLite 数据访问层
│   ├── migrations/
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── pages/          # 路由页面
│   │   ├── components/     # 共享组件
│   │   ├── hooks/          # 自定义 hooks
│   │   ├── api/            # HTTP 请求层
│   │   └── App.tsx
│   ├── index.html
│   ├── package.json
│   └── vite.config.ts
├── container-wrapper/
│   ├── src/server.js       # HTTP wrapper 入口
│   ├── package.json
│   └── Dockerfile.template # 模板 Dockerfile（Go 生成）
├── docs/
│   ├── architecture.md
│   └── plan.md
├── docker-compose.yml
└── README.md
```

## 依赖清单

### Go
| 包 | 用途 |
|---|---|
| github.com/gin-gonic/gin | HTTP 路由 |
| github.com/gorilla/websocket | WebSocket |
| github.com/mattn/go-sqlite3 | SQLite 驱动 |
| github.com/docker/docker | Docker SDK |
| github.com/go-git/go-git | Git 仓库操作 |
| github.com/google/uuid | ID 生成 |

### Node.js (容器内)
| 包 | 用途 |
|---|---|
| @earendil-works/pi-agent-core | Agent SDK |
| @earendil-works/pi-ai | LLM API |
| express | HTTP 框架 |
| typebox | Tool 参数校验 |

### React
| 包 | 用途 |
|---|---|
| react-router-dom | 路由 |
| @monaco-editor/react | JSON/代码编辑器 |
| react-markdown | Markdown 渲染 |
| rehype-highlight | 代码高亮 |
| tailwindcss / antd | UI 样式 |

## 风险与缓解

| 风险 | 缓解 |
|---|---|
| pi npm 包在容器内安装网络问题 | 预构建包含 pi 依赖的 base image |
| Docker SDK API 版本兼容性 | 锁定 Docker Engine 版本，使用稳定 API 版本 |
| LLM API key 在容器内安全隔离 | 通过环境变量注入，不写入镜像层 |
| Tool DSL 表达能力不足 | `javascript` handler 类型兜底，允许内联代码 |
| SSE 代理层内存/连接泄漏 | 设置合理的超时和最大连接数限制 |


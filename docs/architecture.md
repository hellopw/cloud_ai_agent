# Cloud AI Agent — 架构文档

## 摘要

以开源项目 [earendil-works/pi](https://github.com/earendil-works/pi) 为 Agent 运行时，用 Go + React 构建云端的 Agent 管理平台。pi 提供完整的 Agent runtime（tool calling、skills、prompts、session 管理），已有 `@earendil-works/pi-agent-core` 可编程 SDK。需要自建的是：管理后台（Go API + React UI）、Docker 镜像/容器编排层、Tool DSL→TS 代码生成器、以及容器内的 HTTP 桥接层。

## 架构总览

```
┌──────────────────────────────────────────────┐
│              React Frontend (:3000)            │
│  Template Manager | Agent Manager | Chat UI   │
│  Prompt/Skill/Tool Editors                    │
└──────────────────┬───────────────────────────┘
                   │ REST + WebSocket
┌──────────────────┴───────────────────────────┐
│              Go Backend (:8080)                │
│  ┌──────────┐ ┌──────────┐ ┌───────────────┐ │
│  │ CRUD API │ │Docker Svc│ │Session Proxy  │ │
│  │(template │ │(build    │ │(WS↔HTTP/SSE   │ │
│  │ agent    │ │ image    │ │ to container) │ │
│  │ instance │ │ manage)  │ │               │ │
│  │ prompt..)│ │          │ │               │ │
│  └──────────┘ └──────────┘ └───────────────┘ │
│  ┌──────────┐ ┌────────────────────────────┐ │
│  │ Git Repo │ │ SQLite (templates, agents, │ │
│  │ Manager  │ │ instances, prompts, …)     │ │
│  └──────────┘ └────────────────────────────┘ │
└──────────────────┬───────────────────────────┘
                   │ Docker SDK
┌──────────────────┴───────────────────────────┐
│           Docker Daemon (本地)                 │
│  ┌──────────────────────────────────────────┐ │
│  │ Agent Instance Container                 │ │
│  │  Node.js HTTP Wrapper (:动态端口)         │ │
│  │  ├─ POST /chat  → SSE stream            │ │
│  │  ├─ POST /tool-result                   │ │
│  │  ├─ GET  /status                        │ │
│  │  └─ 内部: pi Agent SDK                  │ │
│  │      ├─ skills/  (文件系统加载)          │ │
│  │      ├─ prompts/ (文件系统加载)          │ │
│  │      ├─ tools/   (代码生成→TS 扩展)      │ │
│  │      └─ 代码仓库 (bind mount)            │ │
│  └──────────────────────────────────────────┘ │
└──────────────────────────────────────────────┘
```

## 技术栈

| 层 | 技术 | 说明 |
|---|---|---|
| 前端 | React + TypeScript + Vite | 管理 UI + 嵌入式 Chat |
| 后端 | Go (gin/chi) | REST API + WebSocket 代理 |
| 数据库 | SQLite (mattn/go-sqlite3) | 嵌入式，零配置 |
| 容器 | Docker SDK for Go | 镜像构建、容器生命周期 |
| Agent | pi (@earendil-works/pi-agent-core) | Node.js SDK，容器内运行 |
| 桥接 | Node.js (Express/Fastify) | HTTP wrapper，SSE 流式输出 |

## 三层生命周期

### Agent 模板 (Template)

一个 Dockerfile 定义，绑定了 prompts、skills、tools。

- Dockerfile 基础：`FROM node:22-alpine`，安装 pi 各 npm 包，注入文件系统层的 skills/prompts/tools 扩展
- 用户在管理页创建模板 → 从已定义的 prompts/skills/tools 中选择绑定
- Go 后端负责生成完整构建上下文：Dockerfile + `skills/` + `prompts/` + `extensions/` 目录

```
template-build-context/
├── Dockerfile
├── pi-skills/          # 绑定的 skill.md 文件
├── pi-prompts/         # 绑定的 prompt.md 文件
├── extensions/         # 自动生成的 tool TypeScript 文件
├── package.json        # 依赖 @earendil-works/pi-agent-core
├── server.js           # HTTP wrapper 入口
└── workspace/          # 运行时 bind mount 代码仓库的位置
```

### Agent

模板 + 代码仓库 = 可构建、可运行的 Agent。

- 关联 GitHub/GitLab 仓库地址和分支
- 用户触发构建：Go 拉取代码 + 模板构建上下文 → `docker build` → 本地镜像
- 状态机：`draft` → `building` → `ready` | `failed`
- 构建产物：本地 Docker image tag `cloud-agent/{agent_id}:{version}`

### Agent 实例 (Instance)

用户启动 Agent → 创建容器 → 建立 session → 对话。

- Go 后端调用 Docker SDK：`docker run -d -p {动态端口}:3000 -v {repo_path}:/workspace {image}`
- 容器内 HTTP wrapper 初始化 pi `Agent`，加载 skills/prompts/tools，设置 workspace 为 `/workspace`
- 前端通过 WebSocket 连接 Go 后端，Go 后端代理到容器的 HTTP SSE 端点
- 状态机：`starting` → `running` → `idle` → `stopped` | `error`

## 数据模型

```sql
-- 基础实体
CREATE TABLE prompts (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    content     TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE skills (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    content     TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tools (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    label           TEXT NOT NULL,
    description     TEXT,
    dsl_definition  TEXT NOT NULL,  -- JSON
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 模板：Dockerfile 定义 + 绑定关系
CREATE TABLE templates (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL UNIQUE,
    description         TEXT,
    dockerfile_content  TEXT,       -- 手动定制或自动生成
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE template_prompts (
    template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    prompt_id   TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    PRIMARY KEY (template_id, prompt_id)
);

CREATE TABLE template_skills (
    template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    skill_id    TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (template_id, skill_id)
);

CREATE TABLE template_tools (
    template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    tool_id     TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    PRIMARY KEY (template_id, tool_id)
);

-- Agent：模板 + 代码仓库
CREATE TABLE agents (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    template_id TEXT NOT NULL REFERENCES templates(id),
    repo_url    TEXT NOT NULL,
    branch      TEXT NOT NULL DEFAULT 'main',
    image_tag   TEXT,
    status      TEXT NOT NULL DEFAULT 'draft',  -- draft|building|ready|failed
    error_msg   TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 实例：运行中的 Agent 容器
CREATE TABLE instances (
    id           TEXT PRIMARY KEY,
    agent_id     TEXT NOT NULL REFERENCES agents(id),
    container_id TEXT,
    host_port    INTEGER,
    status       TEXT NOT NULL DEFAULT 'starting',  -- starting|running|idle|stopped|error
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 会话与消息（可选，初期可依赖 pi 自身 session 存储）
CREATE TABLE chat_sessions (
    id          TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL REFERENCES instances(id),
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chat_messages (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role        TEXT NOT NULL,  -- user|assistant|tool
    content     TEXT,
    tool_calls  TEXT,           -- JSON
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Tool DSL 设计

用户在前端以 JSON 形式定义 tool。构建 Agent 时，Go 后端将其编译为 TypeScript 扩展代码。

### DSL Schema

```json
{
  "name": "web_search",
  "label": "Web Search",
  "description": "Search the web using an external API",
  "parameters": {
    "query": { "type": "string", "description": "The search query" },
    "limit":  { "type": "number", "description": "Max results", "default": 10 }
  },
  "handler": {
    "type": "http",
    "method": "GET",
    "url": "https://api.search.com?q={{query}}&limit={{limit}}",
    "headers": {
      "Authorization": "Bearer {{env.SEARCH_API_KEY}}"
    }
  }
}
```

### Handler 类型

| type | 说明 | 关键字段 |
|---|---|---|
| `http` | HTTP 请求 | method, url, headers, body |
| `shell` | 在容器内执行命令 | command, timeout_ms |
| `javascript` | 内联 JS 逻辑（高级） | code |

### 模板变量

- `{{param_name}}` — 替换为 tool 调用时的入参
- `{{env.VAR_NAME}}` — 替换为容器环境变量（用于 API key 等敏感信息）

### 代码生成示例

DSL → Go 模板引擎 → 生成 `.ts` 文件：

```typescript
// auto-generated: web_search.ts
import type { ExtensionContext, ToolDefinition } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

export function activate(context: ExtensionContext) {
  context.registerTool({
    name: "web_search",
    label: "Web Search",
    description: "Search the web using an external API",
    parameters: Type.Object({
      query: Type.String({ description: "The search query" }),
      limit: Type.Number({ description: "Max results", default: 10 }),
    }),
    async execute(args, { signal }) {
      const url = `https://api.search.com?q=${encodeURIComponent(args.query)}&limit=${args.limit}`;
      const resp = await fetch(url, {
        headers: { Authorization: `Bearer ${process.env.SEARCH_API_KEY}` },
        signal,
      });
      return { content: [{ type: "text", text: await resp.text() }] };
    },
  });
}
```

## 容器内 HTTP Wrapper

轻量 Node.js 服务，封装 `@earendil-works/pi-agent-core` 的 `Agent` 类。

### 端点

| Method | Path | 说明 |
|---|---|---|
| POST | `/chat` | 发送用户消息，SSE 流式返回 agent 响应 |
| POST | `/tool-result` | 提交 tool 执行结果（用于需要确认的 tool） |
| POST | `/abort` | 中止当前正在执行的 agent 轮次 |
| GET | `/status` | 返回 agent 当前状态 |

### POST /chat SSE 事件类型

```
event: text_delta
data: {"delta": "Hello, I'll help you..."}

event: tool_call
data: {"toolCallId": "call_1", "toolName": "read", "input": {"path": "/workspace/main.go"}}

event: tool_result
data: {"toolCallId": "call_1", "content": [{"type": "text", "text": "..."}], "isError": false}

event: agent_end
data: {"stopReason": "end_turn", "usage": {"input": 150, "output": 80}}

event: error
data: {"message": "API rate limit exceeded"}
```

### 核心逻辑

```javascript
import { Agent } from "@earendil-works/pi-agent-core";
import { loadSkills } from "@earendil-works/pi-agent-core/skills";
import { loadPromptTemplates } from "@earendil-works/pi-agent-core/prompt-templates";
import { loadExtensions } from "./extensions/loader.js";

const skills = await loadSkills(env, "/app/pi-skills");
const prompts = await loadPromptTemplates(env, "/app/pi-prompts");
const tools = await loadExtensions("/app/extensions");

const agent = new Agent({
  tools,
  sessionId: sessionId,
  streamFn: streamSimple,
  transport: "http",
});

app.post("/chat", async (req, res) => {
  res.setHeader("Content-Type", "text/event-stream");
  agent.on("event", (event) => {
    res.write(`event: ${event.type}\ndata: ${JSON.stringify(event)}\n\n`);
  });
  await agent.prompt(req.body.message);
});
```

## 数据流：一次完整对话

```
User (Browser)                    Go Backend                 Docker Container
     │                                │                            │
     │── WS: { message: "hi" } ──────>│                            │
     │                                │── POST /chat ────────────>│
     │                                │                            │── agent.prompt("hi")
     │                                │                            │── SSE: text_delta
     │                                │<── SSE: text_delta ────────│
     │<── WS: text_delta ─────────────│                            │
     │                                │                            │── agent 调用 tool
     │                                │<── SSE: tool_call ─────────│
     │<── WS: tool_call ──────────────│                            │
     │                                │                            │── tool 执行完毕
     │                                │<── SSE: tool_result ───────│
     │<── WS: tool_result ────────────│                            │
     │                                │<── SSE: agent_end ─────────│
     │<── WS: agent_end ──────────────│                            │
```

## 关键假设

- pi 以 **SDK 模式**嵌入容器内的 Node.js HTTP wrapper，不依赖 pi CLI 的 TUI
- **本地 Docker daemon** 可用且 Go 后端可直接调用 Docker API
- **单用户/单机** 部署场景（初期），暂不涉及多租户、K8s、分布式调度
- 代码仓库仅支持 **GitHub/GitLab HTTPS clone**（暂不需要 SSH 密钥管理）
- LLM API key 通过环境变量注入容器，不在前端暴露

## 待后续讨论

- 认证授权方案（当前按本地 dev tool 处理，暂不加 JWT/OAuth）
- 多 Agent 协作场景（多个实例之间如何通信）
- Agent 资源限制（CPU/内存/timeout）
- 镜像仓库管理（是否推送到远程 registry 或仅本地存储）
- Tool 执行沙箱（是否需要限制 tool 的网络/文件系统访问）


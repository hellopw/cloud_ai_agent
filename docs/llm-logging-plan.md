# LLM Request/Response 日志记录方案

## 目标

将容器内 Agent 的每一次 LLM API 调用（请求/响应）写入文件，文件目录通过挂载卷暴露给 Backend 服务，支持按容器和 Session（Instance）区分。用于排查 system prompt、tools、skills 是否生效。

---

## 架构概览

```
Backend (Go, port 8080)
    │
    │ docker run -v <logDir>:/logs -e INSTANCE_ID=xxx -e LOGS_DIR=/logs
    │ 挂载: builds/<agentID>/llm-logs/  -> 容器内 /logs/
    v
Agent Container (Node.js, port 3000)
    │
    │ LLM 调用前:     写入 /logs/<instanceID>/request-<seq>.jsonl
    │ LLM 调用过程中: 逐行追加 /logs/<instanceID>/response-<seq>.jsonl (SSE events)
    │ 启动时:         写入 /logs/<instanceID>/tools.json
    v
LLM API (Anthropic / OpenAI / custom)
```

**目录结构:**
```
builds/<agentID>/
├── llm-logs/                      <- 新增，挂载到容器 /logs
│   └── <instanceID>/
│       ├── tools.json             <- 启动时写入可用工具列表
│       ├── prompts.json           <- 启动时写入加载的 prompts
│       ├── skills.json            <- 启动时写入加载的 skills
│       ├── request-0001.jsonl     <- 第 1 次 LLM 请求
│       ├── response-0001.jsonl    <- 第 1 次 LLM 响应（逐行 SSE events）
│       ├── request-0002.jsonl
│       ├── response-0002.jsonl
│       └── ...
├── repo/                          <- Git 仓库（已有）
└── build.log                      <- 构建日志（已有）
```

**Backend API:**
- `GET /api/instances/<id>/llm-logs` — 列出该 Instance 的所有日志文件
- `GET /api/instances/<id>/llm-logs/<filename>` — 读取指定日志文件内容

**JSONL 响应格式（每行一个完整 JSON）:**
```jsonl
{"type":"message_start","ts":"2026-06-12T10:30:00.000Z","model":"claude-sonnet-4-20250514","provider":"anthropic"}
{"type":"text_delta","ts":"2026-06-12T10:30:01.000Z","delta":"我来"}
{"type":"text_delta","ts":"2026-06-12T10:30:01.200Z","delta":"分析一下"}
{"type":"tool_use","ts":"2026-06-12T10:30:02.000Z","name":"read_file","input":{"path":"..."}}
{"type":"tool_result","ts":"2026-06-12T10:30:03.000Z","content":"..."}
{"type":"text_delta","ts":"2026-06-12T10:30:04.000Z","delta":"完成了"}
{"type":"message_stop","ts":"2026-06-12T10:30:05.000Z","stop_reason":"end_turn","usage":{"input_tokens":1500,"output_tokens":800}}
```

选择 JSONL 追加而非一次性写入的原因：
1. **实时可见** — `tail -f` 可见 LLM 吐出内容
2. **容错** — agent 崩溃也不会丢失已写入的 events
3. **轻量** — 不需要在内存中积攒整个 events 数组

---

## SSE 响应记录策略（按 Agent 类型）

### Claude Code (`server-claude-code.js`) — SDK 调用点手动记录

有明确的 `client.messages.stream()` 调用点，不依赖 fetch monkey-patch。

- **请求记录**: 调用 `stream()` 前，写入 messages、system prompt、tools、model 参数到 `request-N.jsonl`
- **响应记录**: 在 `for await (const event of stream)` 循环中，每收到一个 event 就 `fs.appendFileSync` 追加一行 JSON 到 `response-N.jsonl`
- **结束时**: 追加 `message_stop` 行，包含 `stop_reason` 和 `usage`

### Codex (`server-codex.js`) — SDK 调用点手动记录

有明确的 `client.responses.create()` 调用点。

- **请求记录**: 调用前写入 input、model 参数
- **响应记录**: 在 `for await (const event of stream)` 循环中逐行追加
- **结束时**: 追加 `agent_end` 行

### PI (`server.js`) — fetch monkey-patch

`@earendil-works/pi-ai` 内部发 HTTP，无直接 SDK 调用点。保持 fetch monkey-patch，但改进为追加写入。

- **请求记录**: intercept 到 LLM API 的 fetch 调用时，写入请求 body
- **响应记录**: 检测 `content-type: text/event-stream`，用 `clone().text()` 等流结束后拿到全量 SSE 文本，按行 split 后逐行写入 JSONL
- **限制**: fetch patch 方式只能等流结束才能写入，但文件内容仍是 JSONL 格式
- 判断 LLM 调用的方式：检查 URL 是否匹配 model 的 baseUrl

### Team (`team-server.js`) — 同 PI

内部使用 PI agent，fetch monkey-patch 方式与 `server.js` 相同。
子目录区分: `/<instanceID>/leader/` 和 `/<instanceID>/<workerName>/`

---

## 实施步骤

### Step 1: 创建共享的 LLM 日志模块 `llm-logger.js`

**文件:** `container-wrapper/src/llm-logger.js`

提供 `createLLMLogger(logsDir, instanceId)` 函数。所有 wrapper 共用。

**核心功能:**
- `newSession()` — 创建日志目录，重置序列号为 0001
- `nextSeq()` — 返回格式化的序号字符串（如 "0001"），递增计数器
- `logRequest(seq, data)` — 写入 `request-<seq>.jsonl`
- `appendResponseLine(seq, event)` — 追加一行 JSON 到 `response-<seq>.jsonl`
- `writeToolsSnapshot(tools)` — 写入 `tools.json`
- `writePromptsSnapshot(promptsDir)` — 扫描目录，写入 `prompts.json`
- `writeSkillsSnapshot(skillsDir)` — 扫描目录，写入 `skills.json`
- `sanitize(obj)` — 递归脱敏，替换 `api_key`、`x-api-key`、`authorization`、`apiKey` 字段为 `***REDACTED***`

---

### Step 2: 修改四个 Wrapper 脚本

#### 2a. `server.js` (PI Agent)

引入 `llm-logger.js`，在 `/chat` handler 开始时 `newSession()`，修改 fetch patch 追加写入文件。

#### 2b. `server-claude-code.js` (Claude Code Agent)

引入 `llm-logger.js`，在 `/chat` handler 开始时 `newSession()`。在 `client.messages.stream()` 调用前后手动记录：调用前写入 request JSONL，stream 迭代中逐行追加 response JSONL。不再依赖 fetch patch 做日志。

#### 2c. `server-codex.js` (Codex Agent)

引入 `llm-logger.js`，在 `/chat` handler 开始时 `newSession()`。在 `client.responses.create()` 调用前后手动记录。

#### 2d. `team-server.js` (Team Agent)

引入 `llm-logger.js`，为 leader 和每个 worker 创建独立 logger（子目录）。添加 `globalThis.fetch` monkey-patch（与 PI 相同的策略）。

---

### Step 3: Backend Go 变更

#### 3a. Docker 服务 — 新增日志目录挂载

**文件:** `backend/internal/docker/service.go`

`RunContainer` 新增 `logDir` 参数，添加 `-v <logDir>:/logs` 挂载。

#### 3b. Agent 服务 — 创建日志目录并传递 env

**文件:** `backend/internal/service/agent.go`

在 `StartInstance` 和 `StartTeamInstance` 中:
1. 创建日志目录 `builds/<agentID>/llm-logs/<instanceID>/` (mode 0777)
2. 将 `INSTANCE_ID` 和 `LOGS_DIR=/logs` 加入容器环境变量

#### 3c. API 路由 — 新增日志查看接口

**文件:** `backend/internal/api/router.go`

新增路由:
- `GET /api/instances/<id>/llm-logs` → `listLLMLogs`
- `GET /api/instances/<id>/llm-logs/<filename>` → `getLLMLogFile`

#### 3d. API Handler — 日志文件读写

**文件:** `backend/internal/api/handlers.go`

- `listLLMLogs`: 读取目录返回文件列表（名称 + 大小），按名称排序
- `getLLMLogFile`: 校验文件名防路径穿越，返回文件内容

---

### Step 4: Dockerfile 模板 — 创建 /logs 目录

所有 Dockerfile 模板添加 `RUN mkdir -p /logs`。

---

### Step 5: 安全措施

**脱敏:** `sanitize()` 递归扫描对象，替换 `api_key`/`apiKey`/`authorization`/`x-api-key` 值为 `***REDACTED***`。
**路径穿越防护:** 文件名仅允许 `[a-zA-Z0-9_.-]+`，用 `filepath.Base()` 清理，确认最终路径前缀。

---

## 涉及文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `container-wrapper/src/llm-logger.js` | **新建** | 共享 LLM 日志模块 |
| `container-wrapper/src/llm-logger.test.js` | **新建** | 单元测试 |
| `container-wrapper/src/server.js` | 修改 | PI agent 日志集成 |
| `container-wrapper/src/server-claude-code.js` | 修改 | Claude Code agent 日志集成 |
| `container-wrapper/src/server-codex.js` | 修改 | Codex agent 日志集成 |
| `container-wrapper/src/team-server.js` | 修改 | Team agent 日志集成 |
| `container-wrapper/dockerfiles/pi.dockerfile` | 修改 | 添加 /logs 目录 |
| `container-wrapper/dockerfiles/claude-code.dockerfile` | 修改 | 添加 /logs 目录 |
| `container-wrapper/dockerfiles/codex.dockerfile` | 修改 | 添加 /logs 目录 |
| `container-wrapper/dockerfiles/team.dockerfile` | 修改 | 添加 /logs 目录 |
| `backend/internal/codegen/dockerfile_types.go` | 修改 | 更新内联 Dockerfile 模板 |
| `backend/internal/docker/service.go` | 修改 | RunContainer 新增 logDir 参数和挂载 |
| `backend/internal/service/agent.go` | 修改 | 创建日志目录，传递 env vars |
| `backend/internal/api/router.go` | 修改 | 新增 llm-logs 路由 |
| `backend/internal/api/handlers.go` | 修改 | 新增日志读取 handler |

---

## 测试方案

### 第一层：`llm-logger.js` 单元测试（必须）

用 Node 内置 `node:test` + `node:assert`，零额外依赖。

```
cd container-wrapper
node --test src/llm-logger.test.js
```

| 测试用例 | 验证内容 |
|---------|---------|
| `newSession` 创建目录 | `createLLMLogger('/tmp/logs', 'inst-123').newSession()` 后目录存在 |
| `newSession` 重置序列号 | 连续两次 `nextSeq()` 返回 `"0001"`, `"0002"`；`newSession()` 后再 `nextSeq()` 回到 `"0001"` |
| `logRequest` 写入文件 | 写入后文件存在，内容为合法 JSONL（一行一个 JSON） |
| `logRequest` 序号格式 | `logRequest('0005', data)` 生成文件 `request-0005.jsonl` |
| `appendResponseLine` 追加 | 调用 3 次，文件有 3 行，每行为合法 JSON |
| `appendResponseLine` 新增文件 | 第一次调用时自动创建 response 文件 |
| `sanitize` API Key | `{api_key: "sk-xxx"}` → `{api_key: "***REDACTED***"}` |
| `sanitize` Authorization | `{headers: {authorization: "Bearer sk-xxx"}}` → authorization 脱敏 |
| `sanitize` x-api-key | `{"x-api-key": "secret"}` → 脱敏 |
| `sanitize` apiKey (camelCase) | `{apiKey: "secret"}` → 脱敏 |
| `sanitize` 嵌套对象 | 深度嵌套的 key 也被脱敏 |
| `sanitize` 数组中的对象 | `[{api_key: "x"}]` → 脱敏 |
| `sanitize` 非敏感字段不改 | `{model: "claude"}` 保持不变 |
| `writeToolsSnapshot` | 写入的 tools.json 包含所有 tool 的 name + input_schema/parameters |
| `writePromptsSnapshot` | 扫描目录，写入 prompts.json，包含文件列表和内容摘要 |
| `writeSkillsSnapshot` | 扫描目录，写入 skills.json，包含文件列表和内容摘要 |
| 并发写入 | 同时调多次 `appendResponseLine`，文件行数正确，不乱序 |
| 空 data 不崩溃 | `logRequest('0001', null)` 写入空对象 `{}` |
| 目录不存在自动创建 | logsDir 的父目录不存在时，`newSession()` 自动创建（mkdirSync recursive） |

### 第二层：Wrapper 集成测试（推荐）

Mock SDK 返回预设 SSE stream，用 `supertest` 调用 Express app 的 `/chat` 端点。

| 测试 | 验证内容 |
|------|---------|
| Claude Code /chat | mock `client.messages.stream()` 返回含 tool_use 的 stream，验证生成 request/response JSONL |
| Codex /chat | mock `client.responses.create()` 返回 stream，验证日志文件写入 |
| 多轮 tool call | mock 3 轮 tool use 循环，验证每轮各生成一对文件 |
| 异常处理 | mock SDK 抛出异常，验证 response 文件记录了 error |
| Chat 重置序列号 | 两次 /chat 请求各自从 0001 开始编号 |

### 第三层：手动冒烟验证

| 步骤 | 检查项 |
|------|--------|
| 启动 agent 实例 | `builds/<agentID>/llm-logs/<instanceID>/tools.json` 存在 |
| | `prompts.json` / `skills.json` 存在（如配置了 prompts/skills） |
| 发送 "hello" | `request-0001.jsonl` 和 `response-0001.jsonl` 生成 |
| 发送带 tool 的请求 | response JSONL 中包含 `tool_use` / `tool_result` 行 |
| API 接口 | `GET /api/instances/<id>/llm-logs` 返回文件列表 |
| | `GET /api/instances/<id>/llm-logs/request-0001.jsonl` 返回内容 |
| 脱敏 | `grep -ri "sk-" builds/<agentID>/llm-logs/` 无匹配 |
| 跨 Session 隔离 | 另启一个实例，确认生成在新目录下 |
| 实时性 | `tail -f builds/<agentID>/llm-logs/<instanceID>/response-0001.jsonl` 可见实时追加 |

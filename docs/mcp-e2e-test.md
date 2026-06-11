# MCP 端到端验证测试文档

## 概述

本文档描述如何完整验证 Cloud AI Agent 平台的 MCP (Model Context Protocol) 工具链，从创建 MCP 工具定义到 Agent 构建、容器启动、最终通过 Chat 接口真实调用 MCP 服务器工具的全流程。

## 测试架构

```
Chat API -> Agent Container (server.js)
  -> Extension (generated .ts) -> mcp-client.js
    -> stdio spawn -> MCP Test Server (mcp_test_server.js)
      -> echo/add tools -> JSON-RPC response
```

## 前提条件

- 测试服务器 10.19.12.152 已部署 cloud_ai_agent（端口 29080）
- Docker 可用（`docker info` 正常）
- 测试用 Git 仓库已创建并包含 MCP 测试服务器脚本

## 第一步：准备 MCP 测试服务器

在测试 Git 仓库的 `src/` 目录下放置 `mcp_test_server.js`（Node.js ESM 脚本，零依赖），实现 MCP stdio 协议，暴露两个工具：

- `echo` — 回显输入消息
- `add` — 两数相加

验证脚本可独立运行：

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | node src/mcp_test_server.js
```

预期输出：

```json
{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"mcp-test-server","version":"0.1.0"}}}
```

测试 tools/list：

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | node src/mcp_test_server.js
```

## 第二步：创建 MCP 工具定义

在 Cloud AI Agent 平台创建 3 种 MCP 工具：

### 2.1 MCP stdio 单工具调用（echo）

```bash
curl -s -X POST http://10.19.12.152:29080/api/tools \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mcp_echo",
    "label": "MCP Echo",
    "description": "Call echo tool on MCP test server",
    "dsl_definition": "{\"name\":\"mcp_echo\",\"label\":\"MCP Echo\",\"description\":\"Echo via MCP stdio\",\"parameters\":{\"message\":{\"type\":\"string\",\"description\":\"Message to echo\"}},\"handler\":{\"type\":\"mcp\",\"transport\":\"stdio\",\"command\":\"node\",\"args\":[\"/workspace/src/mcp_test_server.js\"],\"env\":{},\"tool_name\":\"echo\"}}"
  }'
```

### 2.2 MCP stdio 单工具调用（add）

```bash
curl -s -X POST http://10.19.12.152:29080/api/tools \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mcp_add",
    "label": "MCP Add",
    "description": "Call add tool on MCP test server",
    "dsl_definition": "{\"name\":\"mcp_add\",\"label\":\"MCP Add\",\"description\":\"Add two numbers via MCP\",\"parameters\":{\"a\":{\"type\":\"number\",\"description\":\"First number\"},\"b\":{\"type\":\"number\",\"description\":\"Second number\"}},\"handler\":{\"type\":\"mcp\",\"transport\":\"stdio\",\"command\":\"node\",\"args\":[\"/workspace/src/mcp_test_server.js\"],\"env\":{},\"tool_name\":\"add\"}}"
  }'
```

### 2.3 MCP 工具发现

```bash
curl -s -X POST http://10.19.12.152:29080/api/tools \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mcp_list_tools",
    "label": "MCP List Tools",
    "description": "List all tools on MCP test server",
    "dsl_definition": "{\"name\":\"mcp_list_tools\",\"label\":\"MCP List Tools\",\"description\":\"Discover MCP server tools\",\"parameters\":{},\"handler\":{\"type\":\"mcp\",\"transport\":\"stdio\",\"command\":\"node\",\"args\":[\"/workspace/src/mcp_test_server.js\"],\"env\":{}}}"
  }'
```

## 第三步：创建绑定了 MCP 工具的 Template

创建一个使用默认 Dockerfile（自动包含 mcp-client.js）的模板，并绑定上述 MCP 工具。

先获取上一步创建的工具 ID：

```bash
curl -s http://10.19.12.152:29080/api/tools | \
  python3 -c "
import sys,json
tools=json.load(sys.stdin)
for t in tools:
    if t['name'].startswith('mcp_'):
        print(t['id'], t['name'])
"
```

### 3.1 创建模板

```bash
curl -s -X POST http://10.19.12.152:29080/api/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mcp-test-template",
    "description": "Template for MCP end-to-end testing",
    "dockerfile_content": ""
  }'
```

记录返回的 `template_id`。

### 3.2 绑定 MCP 工具

```bash
TEMPLATE_ID="<上一步返回的 template_id>"
TOOL_ECHO_ID="<mcp_echo 的 id>"
TOOL_ADD_ID="<mcp_add 的 id>"
TOOL_LIST_ID="<mcp_list_tools 的 id>"

curl -s -X PUT "http://10.19.12.152:29080/api/templates/$TEMPLATE_ID/bind" \
  -H "Content-Type: application/json" \
  -d "{\"tool_ids\":[\"$TOOL_ECHO_ID\",\"$TOOL_ADD_ID\",\"$TOOL_LIST_ID\"]}"
```

## 第四步：创建 Agent 并构建

### 4.1 创建 Agent

```bash
curl -s -X POST http://10.19.12.152:29080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mcp-e2e-test",
    "template_id": "<TEMPLATE_ID>",
    "repo_url": "https://code.sohuno.com/binwang219962/seatunnel-web-mcp-py.git",
    "branch": "master",
    "git_username": "binwang219962",
    "git_password": "gZqby5fhgcbKbVczimAv"
  }'
```

记录返回的 `agent_id`。

### 4.2 触发构建

```bash
AGENT_ID="<上一步返回的 agent_id>"
curl -s -X POST "http://10.19.12.152:29080/api/agents/$AGENT_ID/build"
```

### 4.3 查看构建状态和日志

```bash
# 查看状态
curl -s "http://10.19.12.152:29080/api/agents/$AGENT_ID" | python3 -c "import sys,json; a=json.load(sys.stdin); print(f'Status: {a[\"status\"]}')"

# 查看构建日志（构建完成后）
curl -s "http://10.19.12.152:29080/api/agents/$AGENT_ID/log"
```

状态流转：`building` → `ready`（成功）或 `failed`（失败）。

## 第五步：启动实例

### 5.1 检查 Agent 状态为 ready

```bash
curl -s "http://10.19.12.152:29080/api/agents/$AGENT_ID" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])"
# 预期: ready
```

### 5.2 启动实例

```bash
curl -s -X POST "http://10.19.12.152:29080/api/agents/$AGENT_ID/start"
```

记录返回的实例 `id` 和 `host_port`。

### 5.3 检查实例状态

```bash
INSTANCE_ID="<返回的 instance id>"
curl -s "http://10.19.12.152:29080/api/instances/$INSTANCE_ID" | python3 -c "import sys,json; i=json.load(sys.stdin); print(f'Status: {i[\"status\"]}, Port: {i[\"host_port\"]}')"
# 预期: Status: running
```

## 第六步：Chat 对话验证 MCP 调用

### 6.1 验证 echo 工具

通过 Chat 接口发送消息，让 Agent 调用 `mcp_echo` 工具：

```bash
INSTANCE_PORT="<返回的 host_port>"

curl -s -N -X POST "http://10.19.12.152:29080/api/instances/$INSTANCE_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{"message":"请使用 mcp_echo 工具，发送消息 \"Hello MCP E2E Test\""}'
```

预期：SSE 流式响应中应包含 echo 工具调用结果 `Echo: Hello MCP E2E Test`。

### 6.2 验证 add 工具

```bash
curl -s -N -X POST "http://10.19.12.152:29080/api/instances/$INSTANCE_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{"message":"请使用 mcp_add 工具计算 123 + 456 的结果"}'
```

预期：SSE 流式响应中应包含计算结果 `123 + 456 = 579`。

### 6.3 验证工具发现

```bash
curl -s -N -X POST "http://10.19.12.152:29080/api/instances/$INSTANCE_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{"message":"请使用 mcp_list_tools 工具列出所有可用的 MCP 工具"}'
```

预期：响应中应包含 `echo`、`add` 两个工具的信息。

## 验证清单

| 检查项 | 验证方法 | 预期结果 |
|---|---|---|
| 工具创建 | `GET /api/tools` | 包含 `mcp_echo`、`mcp_add`、`mcp_list_tools` |
| 模板绑定 | `GET /api/templates/:id` | `tool_ids` 包含 3 个 MCP 工具 ID |
| Agent 构建 | `GET /api/agents/:id` | `status == "ready"` |
| Dockerfile 含 mcp-client | 构建日志 | Dockerfile 包含 `COPY mcp-client.js` |
| 实例启动 | `GET /api/instances/:id` | `status == "running"` |
| MCP echo 调用 | Chat 接口 | 返回 `Echo: Hello MCP E2E Test` |
| MCP add 调用 | Chat 接口 | 返回 `123 + 456 = 579` |
| MCP discover 调用 | Chat 接口 | 返回 echo、add 工具列表 |

## 故障排查

### 构建失败

```bash
# 查看构建日志
curl -s "http://10.19.12.152:29080/api/agents/$AGENT_ID/log"

# 直接查看文件
ssh 152-binwang "cat ~/cloud_ai_agent/builds/$AGENT_ID/build.log"
```

常见问题：

- **Dockerfile 缺少 mcp-client COPY**：检查 Template 的 `dockerfile_content` 是否为空（空则用默认 Dockerfile，包含 McpClient）
- **Git clone 失败**：检查仓库 URL 和认证信息
- **Docker build 失败**：检查 Dockerfile 语法和基础镜像可访问性

### 实例启动失败

```bash
# 检查容器日志
ssh 152-binwang "docker logs cloud-agent-$INSTANCE_ID 2>&1 | tail -50"
```

常见问题：

- **端口冲突**：`host_port` 已被占用，检查 `netstat -tlnp`
- **MCP 客户端未找到**：确认容器内 `/app/mcp-client.js` 存在且可导入
- **MCP 服务器未找到**：确认仓库代码已挂载到 `/workspace`，且 `mcp_test_server.js` 在 `/workspace/src/` 下

### MCP 工具调用失败

```bash
# 进入容器手动测试
ssh 152-binwang "docker exec -it cloud-agent-$INSTANCE_ID sh"

# 测试 MCP 服务器
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | node /workspace/src/mcp_test_server.js

# 测试 mcp-client.js
cd /app && node -e "
import('./mcp-client.js').then(m => {
  m.callMcpTool({
    transport: 'stdio',
    command: 'node',
    args: ['/workspace/src/mcp_test_server.js'],
    toolName: 'echo',
    toolArgs: { message: 'test' },
  }).then(r => console.log(JSON.stringify(r)));
});
"
```

## 性能与稳定性

- 每次工具调用会启动一个新的 MCP 服务器子进程（stdio 模式）
- MCP 服务器进程在工具调用完成后自动终止
- 超时时间：默认 30 秒
- 如需持久连接，建议使用 SSE 传输模式

## 清理

```bash
# 停止并删除实例
curl -s -X DELETE "http://10.19.12.152:29080/api/instances/$INSTANCE_ID"

# 删除 Agent（可选）
curl -s -X DELETE "http://10.19.12.152:29080/api/agents/$AGENT_ID"

# 清理 Docker 镜像
ssh 152-binwang "docker rmi cloud-agent/$AGENT_ID:latest"
```

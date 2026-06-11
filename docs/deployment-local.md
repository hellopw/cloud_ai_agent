# Cloud AI Agent — 本地部署

## 前提条件

| 依赖 | 版本要求 |
|---|---|
| Go | >= 1.18 |
| Node.js | >= 18 |
| npm | >= 9 |
| Git | 任意版本 |
| Docker | 已安装且 daemon 运行中 |

---

## 克隆项目

```bash
git clone git@code.sohuno.com:binwang219962/cloud_ai_agent.git
cd cloud_ai_agent
```

---

## 开发模式启动

### 1. 启动后端

```bash
cd backend
go run ./cmd/server
```

后端默认监听 `:8080`，启动后自动在 `data/cloud_ai_agent.db` 创建 SQLite 数据库并执行 migration。

可选环境变量：

```bash
PORT=9090 DB_PATH=./data/my.db go run ./cmd/server
```

### 2. 启动前端

新开终端：

```bash
cd frontend
npm install     # 首次运行
npm run dev
```

前端 dev server 监听 `:3000`，`vite.config.ts` 中配置了 `/api` 代理到 `localhost:8080`。

### 3. 验证

打开浏览器访问 `http://localhost:3000`，侧边栏应显示 Prompts / Skills / Tools / Templates / Agents / Instances / Models / Resources 导航。

### Windows PowerShell

```powershell
# 终端 1 - 后端
cd D:\Code\cloud_ai_agent\backend
go run ./cmd/server

# 终端 2 - 前端
cd D:\Code\cloud_ai_agent\frontend
npm install
npm run dev
```

---

## Docker Compose 部署

项目根目录提供 `docker-compose.yml`，同时启动前后端：

```bash
cd cloud_ai_agent

# 可选：设置 API key
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...

# 启动
docker compose up -d

# 查看日志
docker compose logs -f

# 停止
docker compose down
```

服务架构：

```
Browser (:3000) -> nginx -> /api -> Go backend (:8080) -> Docker daemon (/var/run/docker.sock)
                      \-> /    -> Vite build 静态文件
```

注意：backend 容器需要挂载 `/var/run/docker.sock` 才能执行 `docker build` 和 `docker run`。

---

## 二进制构建

### Go 后端编译

```bash
cd backend

# Linux (部署用)
GOOS=linux GOARCH=amd64 go build -o cloud-ai-agent ./cmd/server/

# Windows (本地测试)
go build -o cloud-ai-agent.exe ./cmd/server/
```

### 前端构建

```bash
cd frontend
npm run build     # 输出到 dist/
```

产物位于 `frontend/dist/`，包含 `index.html` 和 `assets/` 目录。

### 后端 + 前端一体化

后端在 `main.go` 中内嵌了一个静态文件服务器：

- 如果请求路径不是 `/api` 开头且磁盘上存在对应文件，直接返回
- 如果文件不存在（SPA 路由如 `/agents`），返回 `index.html`

所以部署时只需：

```bash
cd backend
go build -o cloud-ai-agent ./cmd/server/
# 将二进制和 frontend/dist/ 放在同一目录，设置 FRONTEND_DIR=./frontend
```

---

## 环境变量参考

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | 8080 | 后端 HTTP 监听端口 |
| `DB_PATH` | data/cloud_ai_agent.db | SQLite 数据库文件路径 |
| `FRONTEND_DIR` | frontend | 前端静态文件目录（相对或绝对路径） |
| `PROJECT_ROOT` | 自动检测 | builds/ 和 container-wrapper/ 所在的根目录 |
| `AGENT_PROVIDER` | openai-codex | 容器内默认 provider |
| `AGENT_MODEL` | gpt-5.1-codex-max | 容器内默认 model |
| `AGENT_API_KEY` | - | 容器内默认 API key |
| `AGENT_BASE_URL` | - | 容器内默认 base URL |
| `OPENAI_API_KEY` | - | 容器内 openai provider fallback key |
| `ANTHROPIC_API_KEY` | - | 容器内 anthropic provider fallback key |

注意：`AGENT_*` 和 `OPENAI_API_KEY`/`ANTHROPIC_API_KEY` 仅在启动 Agent 实例时作为默认值注入容器。推荐在 Models 页面创建 Provider Config，启动时选择具体配置。

---

## 故障排查

### 端口被占用

```bash
# Linux
fuser 8080/tcp                 # 查看
fuser -k 8080/tcp              # 杀进程

# Windows
netstat -ano | findstr :8080   # 查看
taskkill /PID {pid} /F         # 杀进程
```

### 数据库被锁

SQLite 使用 WAL 模式，删除 `data/cloud_ai_agent.db` 旁边的 `-wal` 和 `-shm` 文件即可恢复。

### 构建失败

每个 Agent 的构建日志独立存储在 `builds/{agent_id}/build.log`，在 Agent 管理页面点击 "View Log" 查看，或直接读取：

```bash
cat builds/{agent_id}/build.log
```

### Docker 不可用

后端直接调用 `docker` 命令，确认 Docker daemon 运行且当前用户有权限：

```bash
docker info  # 应该正常输出
```

### 前端 SPA 路由返回 404

后端会自动将非 `/api` 且不在磁盘上的路径 fallback 到 `index.html`。如果仍有问题，检查 `FRONTEND_DIR` 环境变量是否正确指向前端构建产物目录。


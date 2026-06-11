# Cloud AI Agent — 构建与部署文档

## 项目结构

```
cloud_ai_agent/
├── backend/                    # Go 后端 (API + Agent 编排)
│   ├── cmd/server/main.go      # 入口
│   ├── internal/
│   │   ├── api/                # HTTP handlers + router
│   │   ├── model/              # 数据模型
│   │   ├── service/            # Agent 构建/启动业务流程
│   │   ├── store/              # SQLite 数据访问层
│   │   ├── docker/             # Docker CLI 封装
│   │   ├── git/                # Git CLI 封装
│   │   ├── codegen/            # Dockerfile 生成 + Tool DSL 编译
│   │   └── proxy/              # WebSocket <-> SSE 代理
│   ├── go.mod / go.sum
│   └── Dockerfile              # 后端 Docker 镜像
├── frontend/                   # React 前端 (Vite + TypeScript)
│   ├── src/
│   │   ├── pages/              # 12 个页面组件
│   │   ├── api/client.ts       # 后端 API 客户端
│   │   └── App.tsx             # 路由 + 侧边栏
│   ├── package.json
│   └── vite.config.ts          # dev 时 /api 代理到 localhost:8080
├── container-wrapper/          # 容器内 Agent 运行时
│   ├── src/server.js           # Express HTTP wrapper (pi SDK)
│   └── package.json            # 依赖 @earendil-works/pi-*
├── deploy/                     # 测试环境部署目录 (gitignore)
│   ├── cloud-ai-agent          # Go 二进制
│   ├── frontend/               # 前端构建产物
│   ├── container-wrapper/      # 容器 wrapper
│   ├── start.sh                # 启动脚本
│   └── http-proxy.conf         # npm 代理配置
├── builds/                     # Agent 构建输出 (gitignore)
│   └── {agent_id}/
│       ├── build.log           # 构建日志
│       ├── Dockerfile          # 生成的 Dockerfile
│       ├── server.js           # wrapper 副本
│       ├── repo/               # git clone 的代码仓库
│       ├── pi-prompts/         # 绑定的 prompts
│       ├── pi-skills/          # 绑定的 skills
│       └── extensions/         # 编译的 tool TS 文件
├── docker-compose.yml          # 本地 Docker 编排
├── docs/                       # 文档
└── data/                       # SQLite 数据库 (gitignore)
```

---

## 本地开发

### 前提条件

| 依赖 | 版本要求 |
|---|---|
| Go | >= 1.18 |
| Node.js | >= 18 |
| npm | >= 9 |
| Git | 任意版本 |
| Docker | 已安装且 daemon 运行中 |

### 1. 克隆项目

```bash
git clone git@code.sohuno.com:binwang219962/cloud_ai_agent.git
cd cloud_ai_agent
```

### 2. 启动后端

```bash
cd backend
go run ./cmd/server
```

后端默认监听 `:8080`，启动后自动在 `data/cloud_ai_agent.db` 创建 SQLite 数据库并执行 migration。

可选环境变量：

```bash
PORT=9090 DB_PATH=./data/my.db go run ./cmd/server
```

### 3. 启动前端

新开终端：

```bash
cd frontend
npm install     # 首次运行
npm run dev
```

前端 dev server 监听 `:3000`，`vite.config.ts` 中配置了 `/api` 代理到 `localhost:8080`。

### 4. 验证

打开浏览器访问 `http://localhost:3000`，侧边栏应显示 Prompts / Skills / Tools / Templates / Agents / Instances / Models / Resources 导航。

### 5. Windows PowerShell 本地开发

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

## 本地 Docker Compose 部署

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

注意：`docker-compose.yml` 中的 backend 容器需要挂载 `/var/run/docker.sock` 才能执行 `docker build` 和 `docker run`。

---

## 本地二进制构建

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

## 测试环境部署 (10.19.12.152)

### 目标信息

| 项目 | 值 |
|---|---|
| 主机 | 10.19.12.152 |
| 用户 | binwang |
| SSH 密钥 | ~/.ssh/config 中已配置 (binwang) |
| 部署目录 | /home/binwang/cloud_ai_agent/ 或项目 deploy/ 目录 |
| 端口 | 8080 |

### 部署步骤

#### 1. 本地构建

```powershell
# 在 Windows 本地执行

# 1. 编译 Go 二进制（交叉编译到 Linux）
cd D:\Code\cloud_ai_agent\backend
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o ..\deploy\cloud-ai-agent .\cmd\server\

# 2. 构建前端
cd D:\Code\cloud_ai_agent\frontend
npm run build

# 3. 复制前端产物到 deploy/
Remove-Item -Recurse -Force D:\Code\cloud_ai_agent\deploy\frontend -ErrorAction SilentlyContinue
Copy-Item -Recurse D:\Code\cloud_ai_agent\frontend\dist D:\Code\cloud_ai_agent\deploy\frontend

# 4. 复制 container-wrapper 到 deploy/
Remove-Item -Recurse -Force D:\Code\cloud_ai_agent\deploy\container-wrapper -ErrorAction SilentlyContinue
Copy-Item -Recurse D:\Code\cloud_ai_agent\container-wrapper D:\Code\cloud_ai_agent\deploy\container-wrapper
```

#### 2. 上传到服务器

```powershell
# 使用 SCP 上传 deploy/ 目录（SSH config 中 binwang 密钥已配置）
scp -r D:\Code\cloud_ai_agent\deploy\* binwang@10.19.12.152:/home/binwang/cloud_ai_agent/
```

#### 3. 服务器上启动

```bash
# SSH 到服务器
ssh binwang@10.19.12.152

cd /home/binwang/cloud_ai_agent
chmod +x cloud-ai-agent start.sh
mkdir -p data logs

# 设置环境变量（可选）
export PORT=8080
export DB_PATH=data/cloud_ai_agent.db
export FRONTEND_DIR=frontend

# 启动
./start.sh

# 或手动启动
nohup ./cloud-ai-agent > logs/server.log 2>&1 &
```

#### 4. 更新部署

后续更新只需重复步骤 1-2，然后在服务器上重启：

```bash
# 服务器上
cd /home/binwang/cloud_ai_agent
fuser -k 8080/tcp
nohup ./cloud-ai-agent > logs/server.log 2>&1 &
```

#### 5. 验证

```
curl http://10.19.12.152:8080/api/health
# 应返回 {"status":"ok"}
```

浏览器访问 `http://10.19.12.152:8080` 打开管理界面。

### 一键部署脚本 (deploy.sh)

在本地项目根目录创建 `scripts/deploy.sh`：

```bash
#!/bin/bash
set -e

HOST="10.19.12.152"
USER="binwang"
REMOTE_DIR="/home/binwang/cloud_ai_agent"

echo "=== 1. Building Go backend ==="
cd backend
GOOS=linux GOARCH=amd64 go build -o ../deploy/cloud-ai-agent ./cmd/server/
cd ..

echo "=== 2. Building frontend ==="
cd frontend && npm run build && cd ..
rm -rf deploy/frontend
cp -r frontend/dist deploy/frontend

echo "=== 3. Copying container-wrapper ==="
rm -rf deploy/container-wrapper
cp -r container-wrapper deploy/container-wrapper

echo "=== 4. Uploading to $HOST ==="
scp -r deploy/* ${USER}@${HOST}:${REMOTE_DIR}/

echo "=== 5. Restarting service ==="
ssh ${USER}@${HOST} "cd ${REMOTE_DIR} && fuser -k 8080/tcp 2>/dev/null; nohup ./cloud-ai-agent > logs/server.log 2>&1 & sleep 1; curl -s http://localhost:8080/api/health"

echo "=== Done ==="
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

### 测试环境特有

测试环境的 Go 二进制已编译了 SSH 密钥路径和 npm 代理配置（硬编码在 `git/service.go` 和 `dockerfile.go` 中）：

- Git SSH：通过宿主机 `~/.ssh/config` 中配置的 binwang 密钥
- Git Clone：`code.sohuno.com` HTTPS 自动转 SSH
- npm 代理：`10.18.34.194:3128`（构建 Docker 镜像时在 Dockerfile 中配置）
- Docker 基础镜像：`private-registry.sohucs.com/domeos-pub/node:22.14.0-alpine3.21`

---

## 构建产物清单

| 产物 | 路径 | 说明 |
|---|---|---|
| Go 二进制 | `backend/cloud-ai-agent` (Linux) / `.exe` (Windows) | API 服务 + 静态文件服务 |
| 前端产物 | `frontend/dist/` | Vite 构建的静态文件 |
| Wrapper 脚本 | `container-wrapper/src/server.js` | Agent 容器内 HTTP 服务 |
| 数据库 | `data/cloud_ai_agent.db` | 自动创建，无需手动初始化 |
| Agent 构建 | `builds/{agent_id}/` | 每次构建生成，含 Dockerfile + repo + log |

---

## 故障排查

### 端口被占用

```bash
# 查看占用 8080 的进程
# Linux
fuser 8080/tcp
# Windows
netstat -ano | findstr :8080

# 杀进程
fuser -k 8080/tcp          # Linux
taskkill /PID {pid} /F     # Windows
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

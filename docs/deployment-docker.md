# Cloud AI Agent — Docker 一体化部署

使用项目根目录 `Dockerfile` 将前后端打包到一个 Nginx 容器中部署。

## 与测试环境部署的区别

| 项目 | 测试环境部署 | Docker 部署 |
|---|---|---|
| 部署方式 | 裸机 Go 二进制 + 静态文件 | Docker 容器 |
| 端口 | 29080 | **29081**（避免冲突） |
| 前端 | Go 服务静态文件 | Nginx 直接托管 |
| API | Go :29080 | Nginx → Go :8080（容器内） |

## 架构

```
浏览器 → host:29081 → Nginx (:80)
                         ├── /          → 前端静态文件
                         └── /api/*     → Go backend (:8080)
```

## 前置准备：内网环境适配

测试环境无法直接访问外网（Docker Hub、Go 模块代理、npm registry 等均不可达），需做以下适配：

### 1. 基础镜像

Docker Hub 不可达，需通过 DaoCloud 镜像拉取基础镜像并 tag 为 Docker Hub 名称：

```bash
# 拉取并 tag 基础镜像
docker pull docker.m.daocloud.io/library/golang:1.25-alpine
docker tag docker.m.daocloud.io/library/golang:1.25-alpine golang:1.25-alpine

docker pull docker.m.daocloud.io/library/node:22-alpine
docker tag docker.m.daocloud.io/library/node:22-alpine node:22-alpine

docker pull docker.m.daocloud.io/library/nginx:1.27-alpine
docker tag docker.m.daocloud.io/library/nginx:1.27-alpine nginx:alpine
```

> **注意**：Go 版本必须与 `backend/go.mod` 中 `go` 指令一致（当前要求 >= 1.25）。

### 2. Docker 代理

Docker daemon 层面配置了内部代理 `http://10.18.34.194:3128`（通过 `docker info` 可查看），但**构建容器不会自动继承**。必须在 `docker build` 时显式传入，且 Dockerfile 中需用 `ARG` + `ENV` 转换：

```dockerfile
FROM golang:1.25-alpine AS go-builder
ARG HTTP_PROXY
ARG HTTPS_PROXY
ENV HTTP_PROXY=$HTTP_PROXY
ENV HTTPS_PROXY=$HTTPS_PROXY
# ... 后续 RUN 指令才能过代理
```

构建命令：

```bash
docker build \
  --build-arg HTTP_PROXY=http://10.18.34.194:3128 \
  --build-arg HTTPS_PROXY=http://10.18.34.194:3128 \
  -t cloud-ai-agent:latest .
```

### 3. Go 模块：vendor 离线编译

代理可以访问外网但会**拦截 GitHub（返回 503）**，且 Go 模块代理 `proxy.golang.org` 不可达。因此必须使用 vendor 模式离线编译：

```bash
# 在能访问外网的开发机上生成 vendor
cd backend
go mod vendor

# 将 vendor 目录同步到服务器
tar czf - vendor/ | ssh <server> "cd /home/binwang/cloud_ai_agent/backend && tar xzf -"
```

Dockerfile 中使用 `-mod=vendor`：

```dockerfile
COPY backend/ .
RUN CGO_ENABLED=0 go build -mod=vendor -o server ./cmd/server/
```

### 4. 前端构建

npm ci / npm run build 可以通过代理正常下载（需正确传入 `ARG` + `ENV`）。

### 5. Alpine 包安装缓慢

`apk add` 安装 `ca-certificates`、`docker`、`git` 等包时通过代理极慢（约 10~15 分钟）。**建议只执行一次**：先构建一个包含依赖的基础镜像，后续构建直接复用。

### 6. 防止 SSH 超时

长时间构建（>2 分钟）会导致 SSH 连接超时中断构建。使用 `nohup` 将构建放到服务器后台执行：

```bash
cd /home/binwang/cloud_ai_agent
nohup docker build \
  --build-arg HTTP_PROXY=http://10.18.34.194:3128 \
  --build-arg HTTPS_PROXY=http://10.18.34.194:3128 \
  -t cloud-ai-agent:latest . \
  > /tmp/docker-build.log 2>&1 &

# 查看进度
tail -f /tmp/docker-build.log
```

## 部署步骤

### 1. 构建镜像（首次）

首次构建需要拉取基础镜像 + 安装 Alpine 依赖包 + Go vendor 编译，耗时较长（15~20 分钟）。

```bash
cd /home/binwang/cloud_ai_agent
nohup docker build \
  --build-arg HTTP_PROXY=http://10.18.34.194:3128 \
  --build-arg HTTPS_PROXY=http://10.18.34.194:3128 \
  -t cloud-ai-agent:latest . \
  > /tmp/docker-build.log 2>&1 &

# 监控构建进度
tail -f /tmp/docker-build.log
```

### 2. 后续更新构建

后续更新（Dockerfile 未变，仅代码变更）时构建较快，因为基础镜像层已缓存：

```bash
cd /home/binwang/cloud_ai_agent
docker build \
  --build-arg HTTP_PROXY=http://10.18.34.194:3128 \
  --build-arg HTTPS_PROXY=http://10.18.34.194:3128 \
  -t cloud-ai-agent:latest .
```

### 3. 启动容器

```bash
docker run -d \
  --name cloud-ai-agent \
  -p 29081:80 \
  -v /home/binwang/cloud_ai_agent/data:/data \
  -e PORT=8080 \
  -e DB_PATH=/data/cloud_ai_agent.db \
  -e FRONTEND_DIR=/usr/share/nginx/html \
  cloud-ai-agent:latest
```

注意：
- `-p 29081:80`：映射到 29081，不与测试环境 29080 冲突
- `-v .../data:/data`：挂载数据目录，保留 SQLite 数据库
- `PORT=8080`：容器内 Go 监听 8080，与 nginx 代理一致

### 4. 验证

```bash
# 健康检查
curl http://localhost:29081/api/health
# 预期: {"status":"ok"}

# 前端页面
curl -s -o /dev/null -w '%{http_code}' http://localhost:29081/
# 预期: 200
```

### 5. 更新部署

```bash
# 重新构建并替换容器
docker build \
  --build-arg HTTP_PROXY=http://10.18.34.194:3128 \
  --build-arg HTTPS_PROXY=http://10.18.34.194:3128 \
  -t cloud-ai-agent:latest .

docker stop cloud-ai-agent
docker rm cloud-ai-agent
# 然后重新执行步骤 3 的 docker run
```

### 6. 查看日志

```bash
docker logs -f cloud-ai-agent
```

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | 8080 | Go 后端监听端口（仅容器内） |
| `DB_PATH` | data/cloud_ai_agent.db | SQLite 数据库路径 |
| `FRONTEND_DIR` | frontend | 前端静态文件目录 |
| `ANTHROPIC_API_KEY` | - | Anthropic API 密钥 |
| `OPENAI_API_KEY` | - | OpenAI API 密钥 |

## 故障排查

### 端口冲突

29081 被占用时换一个端口：

```bash
docker run -d --name cloud-ai-agent -p 29082:80 ... cloud-ai-agent:latest
```

### 数据库文件

如果挂载的数据库目录为空，Go 服务会自动创建 `cloud_ai_agent.db`。首次启动需要手动 seed 数据或从现有部署复制数据库文件：

```bash
cp /home/binwang/cloud_ai_agent/data/cloud_ai_agent.db /home/binwang/cloud_ai_agent/data/backup.db
```

### Docker Hub 不可达

错误信息：`ERROR: failed to do request: Head "https://registry-1.docker.io/v2/...": Service Unavailable`

解决：通过 DaoCloud 镜像拉取并 tag（见"前置准备 → 基础镜像"）。

### Go 版本不匹配

错误信息：`go.mod requires go >= 1.25.5 (running go 1.22.12)`

解决：Dockerfile 中 `FROM golang` 版本必须与 `backend/go.mod` 一致。

### Go 模块下载失败（代理 503）

错误信息：`fatal: unable to access 'https://github.com/...': CONNECT tunnel failed, response 503`

解决：使用 vendor 离线编译（见"前置准备 → Go 模块"）。

### Go 模块代理超时

错误信息：`dial tcp ...:443: i/o timeout`

即使配置了 `GOPROXY=https://goproxy.cn`，外网代理仍不可达。同样使用 vendor 离线编译解决。

### Alpine apk 安装缓慢/超时

`apk add docker` 包体积大，通过内网代理安装需 10~15 分钟，可能触发构建超时。

解决：
1. 使用 `nohup` 后台执行构建
2. 首次构建后该层会被缓存，后续构建无需重复安装
3. 或预先构建包含依赖的基础镜像

### 构建容器无法访问外网

即使 Docker daemon 配置了代理，`docker build` 中的 RUN 指令也不会自动使用。错误现象为所有外网请求（Go 模块、npm、apk）都超时。

解决：必须同时满足两个条件：
1. `docker build --build-arg HTTP_PROXY=... --build-arg HTTPS_PROXY=...`
2. Dockerfile 中声明 `ARG HTTP_PROXY` + `ENV HTTP_PROXY=$HTTP_PROXY`

### SSH 连接超时中断构建

长时间运行的 `docker build`（通过 SSH 执行）会因 SSH 超时而中断。

解决：使用 `nohup ... > /tmp/build.log 2>&1 &` 将构建放到服务器后台，通过 `tail -f /tmp/build.log` 监控。

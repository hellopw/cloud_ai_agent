# Cloud AI Agent — 测试环境部署

## 目标信息

| 项目 | 值 |
|---|---|
| 主机 | 10.19.12.152 |
| 用户 | binwang |
| SSH 别名 | `152-binwang`（`~/.ssh/config` 中已配置） |
| 部署目录 | `/home/binwang/cloud_ai_agent/` |
| 端口 | 29080（默认 8080 被 Tomcat 占用） |

## 部署步骤

### 1. 本地构建

在 Windows 本地执行：

```powershell
cd D:\Code\cloud_ai_agent

# 1. 编译 Go 二进制（交叉编译到 Linux）
# Go build cache 可能因权限问题失败，设置 GOCACHE 到项目内目录
cd backend
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:GOCACHE="D:\Code\cloud_ai_agent\.go-cache"
go build -o ..\deploy\cloud-ai-agent .\cmd\server\

# 2. 构建前端
cd ..\frontend
npm run build

# 3. 复制前端产物到 deploy/
Remove-Item -Recurse -Force ..\deploy\frontend -ErrorAction SilentlyContinue
Copy-Item -Recurse dist ..\deploy\frontend

# 4. 复制 container-wrapper 到 deploy/
Remove-Item -Recurse -Force ..\deploy\container-wrapper -ErrorAction SilentlyContinue
Copy-Item -Recurse ..\container-wrapper ..\deploy\container-wrapper
```

### 2. 上传到服务器

```powershell
# 使用 SSH 别名上传
scp -r D:\Code\cloud_ai_agent\deploy\* 152-binwang:/home/binwang/cloud_ai_agent/
```

常见问题与处理：

- **SCP 上传失败 `dest open ... Failure`**：服务器上旧的 `cloud-ai-agent` 二进制可能正在运行，文件被占用。先 SSH 到服务器删除旧二进制再上传：`ssh 152-binwang "rm -f /home/binwang/cloud_ai_agent/cloud-ai-agent"`
- **SCP 认证失败**：确认 `~/.ssh/config` 中有 `152-binwang` Host 配置，且对应的 `IdentityFile` 密钥存在
- **其他文件（frontend、container-wrapper 等）上传无此问题**：只有正在运行的二进制文件会因为被占用而写入失败

### 3. 服务器上启动

```bash
# SSH 到服务器
ssh 152-binwang

cd /home/binwang/cloud_ai_agent
chmod +x cloud-ai-agent start.sh
mkdir -p data logs

# 启动（指定端口 29080，因为 8080 被 Tomcat 占用）
PORT=29080 nohup ./cloud-ai-agent > logs/server.log 2>&1 &

# 或使用 start.sh（需先确认 start.sh 中端口配置）
```

### 4. 更新部署（重启）

后续更新只需重复步骤 1-2，然后在服务器上重启：

```bash
# 注意：fuser 在测试服务器上不可用，改用 pkill
ssh 152-binwang "cd /home/binwang/cloud_ai_agent && pkill -f cloud-ai-agent; sleep 2; PORT=29080 nohup ./cloud-ai-agent > logs/server.log 2>&1 &"
```

## 环境配置

### deploy/.env

```env
PORT=29080
DB_PATH=data/cloud_ai_agent.db
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
```

### deploy/start.sh

默认端口设为 29080（`${PORT:-29080}`），与服务器上 Tomcat 占用 8080 的情况匹配。

## 测试环境特有配置

Go 二进制中已硬编码以下配置：

- Git SSH：通过宿主机 `~/.ssh/config` 中配置的 binwang 密钥
- Git Clone：`code.sohuno.com` HTTPS 自动转 SSH
- npm 代理：`10.18.34.194:3128`（构建 Docker 镜像时在 Dockerfile 中配置）
- Docker 基础镜像：`private-registry.sohucs.com/domeos-pub/node:22.14.0-alpine3.21`


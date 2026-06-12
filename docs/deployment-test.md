# Cloud AI Agent — 测试环境部署

## 目标信息

| 项目 | 值 |
|---|---|
| 主机 | 10.19.12.152 |
| 用户 | binwang |
| SSH 别名 | `152-binwang`（`~/.ssh/config` 中已配置） |
| 部署目录 | `/home/binwang/cloud_ai_agent/` |
| 端口 | 29080（默认 8080 被 Tomcat 占用） |

## 部署前检查

**关键**：上传前必须清理 `deploy/` 目录下的本地运行时产物，否则 SCP 会覆盖服务器数据：

```powershell
# 清理本地运行时产物（防止覆盖服务器数据）
Remove-Item -Recurse -Force D:\Code\cloud_ai_agent\deploy\data -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force D:\Code\cloud_ai_agent\deploy\logs -ErrorAction SilentlyContinue
Remove-Item -Force D:\Code\cloud_ai_agent\deploy\cloud-ai-agent.exe -ErrorAction SilentlyContinue
```

## 部署步骤

### 1. 本地构建

在 Windows 本地执行：

```powershell
cd D:\Code\cloud_ai_agent

# 0. 清理上次的构建产物
Remove-Item -Force ..\deploy\cloud-ai-agent -ErrorAction SilentlyContinue

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
- **数据覆盖风险**：如果本地 `deploy/data/` 目录下有数据库文件，SCP 上传会直接覆盖服务器上的生产数据库。务必在构建前清理（见上方「部署前检查」）

### 3. 服务器上启动

```bash
# SSH 到服务器
ssh 152-binwang

cd /home/binwang/cloud_ai_agent
chmod +x cloud-ai-agent
mkdir -p data logs

# 推荐使用 start.sh 启动（已配置好端口、日志等）
bash start.sh

# 或手动启动（指定端口 29080，因为 8080 被 Tomcat 占用）
PORT=29080 nohup ./cloud-ai-agent > logs/server.log 2>&1 &
```

注意：
- **start.sh 换行符**：如果 `start.sh` 报 `$'\r': 未找到命令`，说明文件是 Windows CRLF 换行。在服务器上执行 `sed -i 's/\r$//' start.sh` 修复
- **SSH nohup 问题**：直接通过 SSH 执行 `nohup ... &` 可能导致 SSH 等待后台进程无法退出（exit 255）。推荐 SSH 到服务器后用 `bash start.sh` 启动，或使用 `start.sh` 脚本

### 4. 更新部署（重启）

后续更新只需重复步骤 1-2，然后在服务器上重启：

```bash
# 方式一：SSH 到服务器用 start.sh 重启（推荐）
ssh 152-binwang "cd /home/binwang/cloud_ai_agent && pkill -f cloud-ai-agent; sleep 2; bash start.sh"

# 方式二：直接远程执行（注意 pkill 找不到进程时会返回非零但不影响）
ssh 152-binwang "cd /home/binwang/cloud_ai_agent && pkill -f cloud-ai-agent || true; sleep 2; PORT=29080 nohup ./cloud-ai-agent > logs/server.log 2>&1 &"
```

### 5. 验证部署

```bash
# 健康检查
ssh 152-binwang "curl -s http://localhost:29080/api/health"
# 预期返回: {"status":"ok"}

# 查看日志
ssh 152-binwang "tail -20 /home/binwang/cloud_ai_agent/logs/server.log"
# 预期包含: Database migrated successfully / Server starting on :29080 / DB path: data/cloud_ai_agent.db
```

## 常见问题排查

### 数据库被覆盖

**现象**：部署后所有实例、Agent 等数据消失，`/api/instances` 返回 `[]`

**原因**：本地 `deploy/data/` 目录下有残留的空数据库文件，SCP 上传时覆盖了服务器数据库

**解决**：
1. 每次构建前清理 `deploy/data/` 和 `deploy/logs/`
2. 如有重要数据，部署前先备份：`ssh 152-binwang "cp /home/binwang/cloud_ai_agent/data/cloud_ai_agent.db /tmp/backup.db"`

### start.sh 换行符错误

**现象**：启动时报 `$'\r': 未找到命令`、`没有那个文件或目录`

**原因**：Windows 上编辑的 `start.sh` 使用了 CRLF 换行，且 `DB_PATH` 变量行尾的 `\r` 被带入数据库文件名，导致生成 `cloud_ai_agent.db\r` 这种异常文件

**解决**：在服务器上执行 `sed -i 's/\r$//' start.sh` 修复换行

### 前端 j.prompts is undefined

**现象**：前端报 `Uncaught TypeError: can't access property "length", j.prompts is undefined`

**原因**：`/api/instances/:id/config` 返回错误（如 404）时，`.catch(() => {})` 吞掉了错误，`setInstanceConfig(data)` 将状态替换为不含 `prompts` 字段的错误对象

**修复**：`ChatPage.tsx:117` — `setInstanceConfig` 时对 `prompts`/`skills`/`tools` 做 fallback 默认空数组

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

## 容器端口分配

Instance 启动时自动分配宿主机端口（映射容器 3000 端口）：

- **范围**：3001–12000
- **算法**：对 instance UUID 取 FNV-32a hash，映射到 3001 + hash%9000
- **冲突处理**：检测到端口被已有 running/starting 实例占用时自动顺延到下一个可用端口


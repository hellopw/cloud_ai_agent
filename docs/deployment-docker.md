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

## 部署步骤

### 1. 构建镜像

在项目根目录执行：

```powershell
cd D:\Code\cloud_ai_agent
docker build -t cloud-ai-agent:latest .
```

### 2. 启动容器

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

### 3. 验证

```bash
# 健康检查
curl http://localhost:29081/api/health
# 预期: {"status":"ok"}

# 前端页面
curl -s -o /dev/null -w '%{http_code}' http://localhost:29081/
# 预期: 200
```

### 4. 更新部署

```bash
# 重新构建并替换容器
docker build -t cloud-ai-agent:latest .
docker stop cloud-ai-agent
docker rm cloud-ai-agent
# 然后重新执行步骤 2 的 docker run
```

### 5. 查看日志

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

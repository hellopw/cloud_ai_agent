# Cloud AI Agent — 部署验证

部署后必须验证服务正确启动，关键原则：**服务启动需要时间，验证不能太快**。

## 验证步骤

### 1. 确认进程存在

```bash
ssh 152-binwang 'ps aux | grep cloud-ai-agent | grep -v grep'
```

应看到类似输出：

```
binwang  12084  0.0  0.0 1282196 8956 ?  Sl  14:13  0:00 ./cloud-ai-agent
```

如果没有进程输出，说明启动失败，检查日志。

### 2. 检查启动日志

```bash
ssh 152-binwang 'cat /home/binwang/cloud_ai_agent/logs/server.log'
```

正常日志示例：

```
2026/06/11 14:13:45 Database migrated successfully
2026/06/11 14:13:45 Server starting on :29080
2026/06/11 14:13:45 Frontend dir: frontend
2026/06/11 14:13:45 DB path: data/cloud_ai_agent.db
```

异常日志特征：

- `address already in use` — 端口被占用，换端口或 kill 占用进程
- `Failed to start server` — 多种可能，查看具体错误信息
- 无日志输出 — 二进制权限问题，`chmod +x cloud-ai-agent`

### 3. 等待服务就绪后再验证

**关键：启动后等待至少 3-5 秒再验证**。验证过快会收到 Tomcat 或其他服务的 404 响应，误判为失败。

```bash
# 正确做法：先启动，sleep 等待，再验证
ssh 152-binwang 'cd /home/binwang/cloud_ai_agent && PORT=29080 nohup ./cloud-ai-agent > logs/server.log 2>&1 & sleep 5 && curl -s http://localhost:29080/api/health'
```

### 4. 健康检查

```bash
ssh 152-binwang 'curl -s http://localhost:29080/api/health'
```

预期返回：

```json
{"status":"ok"}
```

如果返回 404 页面（尤其是 Tomcat 的 404 页面），可能原因：

- 验证太快，服务还未完成监听绑定，等待后重试
- 端口被其他服务占用，检查 `netstat -tlnp | grep 29080`
- 服务启动时端口绑定失败，检查日志

### 5. 前端页面验证

```bash
ssh 152-binwang 'curl -s --max-time 5 http://localhost:29080/ | head -c 200'
```

预期返回 HTML 文档开头（`<!DOCTYPE html>...`），包含 Vite 构建的资源引用。

### 6. 外部访问验证

```bash
ssh 152-binwang 'curl -s --max-time 5 http://10.19.12.152:29080/api/health'
# 浏览器访问 http://10.19.12.152:29080
```

## 验证检查清单

| 检查项 | 命令 | 预期结果 |
|---|---|---|
| 进程存在 | `ps aux \| grep cloud-ai-agent` | 有 `./cloud-ai-agent` 进程 |
| 端口监听 | `netstat -tlnp \| grep 29080` | `0.0.0.0:29080 ... LISTEN` |
| 日志无报错 | `tail logs/server.log` | 无 `Failed`、`address already in use` |
| API 健康检查 | `curl localhost:29080/api/health` | `{"status":"ok"}` |
| 前端页面 | `curl localhost:29080/` | 返回 HTML（含 `<!DOCTYPE html>`） |
| 外部可访问 | `curl 10.19.12.152:29080/api/health` | `{"status":"ok"}` |

## 常见误判

- **验证太早拿到 Tomcat 404 以为是端口冲突**：实际上是 cloud-ai-agent 还没启动完成，curl 连接被拒绝后 fallback 了。等几秒再试。
- **SCP 上传失败后直接认为验证失败**：二进制文件可能根本没更新成功。先确认 SCP 无报错。
- **日志显示 `address already in use` 但 netstat 看不到进程**：端口可能被其他用户（如 root）的进程占用，需要用 `netstat -tlnp` 查看 PID 确认。


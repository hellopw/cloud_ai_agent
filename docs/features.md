# Cloud AI Agent — 功能总结

## 项目定位

Cloud AI Agent 是一个云端的 AI Agent 全生命周期管理平台。用户可以在 Web UI 中定义 Agent 的提示词(Prompts)、技能(Skills)、工具(Tools)和模板(Templates)，关联 Git 代码仓库，一键构建 Docker 镜像并启动 Agent 实例，通过内置 Chat 界面进行实时对话。

## 核心功能

### 1. Prompt 管理
- 创建/编辑/删除 Prompt 模板（Markdown 格式，含 frontmatter）
- Prompt 可绑定到 Template，在构建时注入容器内的 pi-prompts/ 目录

### 2. Skill 管理
- 创建/编辑/删除 Skill（Markdown 格式，含 name/description frontmatter）
- Skill 可绑定到 Template，在构建时注入容器内的 pi-skills/ 目录

### 3. Tool 管理
- 创建/编辑/删除 Tool（name, label, description, DSL JSON 定义）
- Tool 可绑定到 Template，构建时通过代码生成器编译为 TypeScript 扩展
- Tool DSL 支持 HTTP/shell/javascript 三种 handler 类型

### 4. Template 管理
- 创建/编辑/删除 Template（name, description, 自定义 dockerfile_content）
- 批量绑定 Prompts、Skills、Tools
- 查看 Template 生成的 Dockerfile 内容
- 默认 Dockerfile 基于 `private-registry.sohucs.com/domeos-pub/node:22.14.0-alpine3.21`

### 5. Agent 管理
- 创建 Agent：选择 Template、填写 Git 仓库 URL/分支/认证信息
- 从已保存的 Git Resource 一键填充仓库信息
- 勾选附加资源（Database, Knowledge Base）
- Agent 状态机：draft -> building -> ready | failed
- Build：拉取代码 -> 生成构建上下文 -> Docker build（异步，独立日志文件）
- View Log：查看构建日志
- View Dockerfile：查看生成的 Dockerfile
- Start：选择 Model Config -> 启动容器实例
- 仅 draft/failed 状态可编辑，防止修改已构建的 Agent

### 6. Instance 管理
- 查看所有实例及其状态（running, error, stopped）
- 启动新实例：选择 Ready 状态的 Agent + 选择 Model Config
- Chat 入口：点击进入实时对话界面
- Stop：停止容器并删除实例记录
- 失败实例展示 error_msg

### 7. Model (Provider Config) 管理
- 创建/编辑/删除 Provider 配置
- 支持三种 Provider：openai-codex, anthropic, openai
- 配置项：name, provider, model_id, api_key, base_url
- Model ID 有下拉提示（如 gpt-5.1-codex-max, claude-sonnet-4-6 等）
- api_key 编辑时留空则保留原有值
- 启动 Instance 时选择对应配置

### 8. Resource 管理
- 创建/编辑/删除资源
- 三种资源类型：
  - **Git Repo**: url, username, password/token, branch
  - **Database**: db_type, connection, username, password
  - **Knowledge Base**: api_endpoint, doc_url
- 创建 Agent 时可从已保存的 Git Resource 快速填充仓库信息
- 创建 Agent 时可勾选 database/knowledge 资源关联

### 9. Chat 对话
- WebSocket 连接，实时流式输出
- 支持消息类型：user, assistant, tool, system
- Tool call 卡片展示（tool 名称、参数）
- 流式内容逐字渲染，光标闪烁动画
- 显示当前 provider 和 model 信息
- 连接状态指示

### 10. Agent Team（多 Agent 团队）
- 创建/编辑/删除团队：定义成员、绑定配置
- **主从协作模式**：Leader 负责任务分派，Worker 执行具体任务
- 每个成员独立 LLM 配置（不同 Provider/Model）、独立上下文
- Team 级 Prompts/Skills 覆盖（所有成员共享，成员可额外绑定）
- Build：为每个成员生成独立工具/Prompts/Skills，打包为单容器
- Start：启动团队容器，所有成员运行在同一容器内
- Leader 内置 `delegate_task` 工具：将任务委派给指定 Worker
- 实例列表显示 Team 标识，Chat 自动路由到 Leader

## 容器内 Agent 能力

每个 Agent 实例容器内置 7 个工具：

| 工具 | 功能 |
|---|---|
| bash | 执行 Shell 命令 |
| read_file | 读取文件内容 |
| write_file | 写入文件 |
| edit_file | 结构化编辑文件（find & replace） |
| search_content | grep 搜索 |
| list_files | 列出目录内容 |
| git | 执行 git 命令 |

代码仓库通过 bind mount 挂载到容器的 /workspace，Agent 可直接操作仓库文件。

## 技术特点

- **零配置数据库**：SQLite，纯 Go 实现 (modernc.org/sqlite)，无需 CGO
- **无框架后端**：Go net/http 手动路由，无 gin/chi 等框架依赖
- **CLI 调用**：Docker 和 Git 操作通过 exec.Command 调用 CLI，非 SDK
- **智能 Git Clone**：自动检测 code.sohuno.com 域名，HTTPS 转为 SSH
- **独立构建日志**：每个 Agent 每次构建写入独立 log 文件
- **SSH 密钥复用**：使用宿主机 ~/.ssh/config 中配置的密钥
- **纯 CSS 前端**：无 UI 框架依赖，轻量级

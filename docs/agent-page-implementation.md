# Agent 聚合页实现方案

## 摘要

在现有独立路由的基础上新增 `/agent` 聚合页，通过 Tab 切换统一展示实例（Instances）、镜像（Agents）、团队（Agent Teams）、模板（Templates）四个子页。后端 API 和数据模型保持不变，原有独立路由 `/instances`、`/agents`、`/agent-teams`、`/templates` 全部保留。

## 概念映射

```text
实例 (Instances)  = 运行的容器实例（含 Team 实例） = 镜像/团队 + 模型(ProviderConfig) + 启动(docker run)
镜像 (Agents)     = 单 Agent 容器镜像             = 模板 + Git Repo + Resource → Docker Build → Image Tag
团队 (Teams)      = 多 Agent 容器镜像             = 模板 + Members(Leader+Workers) → Docker Build → Image Tag
模板 (Templates)  = 基础镜像                     = Dockerfile + Prompt/Skill/Tool 绑定
```

## 实现变更

### 新增文件

**`frontend/src/pages/AgentPage.tsx`** — Tab 容器页

- 通过 `URLSearchParams` 管理当前 Tab：`?tab=instances|images|teams|templates`
- 默认选中 `instances`
- 四个 Tab 面板各自直接渲染已有页面组件：
  - 实例 → `<InstancesPage />`
  - 镜像 → `<AgentsPage />`
  - 团队 → `<AgentTeamsPage />`
  - 模板 → `<TemplatesPage />`
- InstancesPage 内的 team_id 标记和 Team 启动区域正常渲染，无需额外处理
- 组件内原有 Link 导航（Edit、New Template、Chat 入口等）跳转到独立路由，符合"保留独立入口"的设计
- Tab 导航栏使用项目现有的 CSS 变量，inline style 实现

### 修改文件

**`frontend/src/App.tsx`**

```diff
+ import AgentPage from './pages/AgentPage'
+ import AgentTeamsPage from './pages/AgentTeamsPage'

  const navItems = [
+   { to: '/agent', label: 'Agent' },
    { to: '/prompts', label: 'Prompts' },
    ...
  ]

  <Routes>
+   <Route path="/agent" element={<AgentPage />} />
    ...
  </Routes>
```

前端路由从 16 条变为 17 条（含 agent-teams 3 条路由）。原有所有路由保持不变。

### Tab 标签定义

| Tab | URL 参数 | 渲染组件 | 说明 |
|-----|---------|---------|------|
| 实例 | `?tab=instances` | `InstancesPage` | 运行的容器实例，含 Team 标识和启动入口 |
| 镜像 | `?tab=images` | `AgentsPage` | 单 Agent 容器镜像，含创建/构建/日志/编辑 |
| 团队 | `?tab=teams` | `AgentTeamsPage` | 多 Agent 团队镜像，含成员管理/Build/Start |
| 模板 | `?tab=templates` | `TemplatesPage` | 基础镜像，含创建/编辑/Dockerfile 预览 |

### 不修改的部分

- 后端 Go 代码（model、store、api、router）
- 前端现有页面组件（InstancesPage、AgentsPage、AgentTeamsPage、TemplatesPage）
- 前端 API 层（`client.ts`，已含 agentTeamsApi）
- InstancesPage 已含 Team 标识和 Team 启动区域，聚合页中直接复用

## 交互行为

- 访问 `/agent` → 默认显示"实例" Tab，URL 自动补全为 `/agent?tab=instances`
- 切换 Tab → URL search params 同步更新，组件重新挂载触发数据重载
- 聚合页 Tab 内点击 `Edit` / `New Template` / `Chat` 等 Link → 跳转到独立路由页面
- 聚合页 Tab 内执行 CRUD 操作（创建、删除、构建、启动等） → 在当前 Tab 内完成，不跳转
- 浏览器前进/后退 → Tab 状态正确恢复

## 测试验证

1. 访问 `/agent`，默认显示"实例"Tab，内容与 `/instances` 一致
2. 实例 Tab 中 Team 实例显示 **Team: {name}** 标签
3. 切换到"镜像"Tab，确认显示 Agents 创建表单和列表
4. 切换到"团队"Tab，确认显示 AgentTeams 列表（含 Build/Start/Edit 按钮）
5. 在"团队"Tab 点击 "Edit" → 正确跳转到 `/agent-teams/:id`
6. 在"团队"Tab 中执行 Build 操作 → 在当前 Tab 内完成，不跳转
7. 切换到"模板"Tab，确认显示 Templates 列表
8. 在"模板"Tab 点击 "Edit" → 正确跳转到 `/templates/:id`
9. 侧边栏同时存在 "Agent" 和 "Agent Teams"/"Agents"/"Instances"/"Templates" 五个入口

## 假设与约束

- 页面组件在 loading 状态返回 `<div className="content">`，Tab 内渲染会产生嵌套 `.content` DOM 结构。项目使用纯 CSS 无组件库，不影响布局和功能，不做额外处理。
- 每次 Tab 切换子组件重新挂载，`useEffect` 重新触发数据请求，这是预期行为。
- 后端无需任何变更。


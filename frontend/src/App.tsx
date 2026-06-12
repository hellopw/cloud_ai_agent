import { Routes, Route, NavLink } from 'react-router-dom'
import PromptsPage from './pages/PromptsPage'
import PromptEditPage from './pages/PromptEditPage'
import SkillsPage from './pages/SkillsPage'
import SkillEditPage from './pages/SkillEditPage'
import ToolsPage from './pages/ToolsPage'
import ToolEditPage from './pages/ToolEditPage'
import MemoriesPage from './pages/MemoriesPage'
import MemoryEditPage from './pages/MemoryEditPage'
import TemplatesPage from './pages/TemplatesPage'
import TemplateEditPage from './pages/TemplateEditPage'
import AgentsPage from './pages/AgentsPage'
import InstancesPage from './pages/InstancesPage'
import ChatPage from './pages/ChatPage'
import ResourcesPage from './pages/ResourcesPage'
import ModelsPage from './pages/ModelsPage'
import AgentTeamsPage from './pages/AgentTeamsPage'
import AgentTeamEditPage from './pages/AgentTeamEditPage'
import AgentPage from './pages/AgentPage'

const navItems = [
  { to: '/agent', label: 'Agent' },
  { to: '/prompts', label: 'Prompts' },
  { to: '/skills', label: 'Skills' },
  { to: '/tools', label: 'Tools' },
  { to: '/memories', label: 'Memories' },
  { to: '/templates', label: 'Templates' },
  { to: '/agents', label: 'Agents' },
  { to: '/agent-teams', label: 'Agent Teams' },
  { to: '/instances', label: 'Instances' },
  { to: '/models', label: 'Models' },
  { to: '/resources', label: 'Resources' },
]

function App() {
  return (
    <div className="app">
      <aside className="sidebar">
        <h1 className="logo">Cloud AI Agent</h1>
        <nav>
          {navItems.map((item) => (
            <NavLink key={item.to} to={item.to} className={({ isActive }) => isActive ? 'nav-active' : ''}>
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <main className="content">
        <Routes>
          <Route path="/" element={<PromptsPage />} />
          <Route path="/agent" element={<AgentPage />} />
          <Route path="/prompts" element={<PromptsPage />} />
          <Route path="/prompts/new" element={<PromptEditPage />} />
          <Route path="/prompts/:id" element={<PromptEditPage />} />
          <Route path="/skills" element={<SkillsPage />} />
          <Route path="/skills/new" element={<SkillEditPage />} />
          <Route path="/skills/:id" element={<SkillEditPage />} />
          <Route path="/tools" element={<ToolsPage />} />
          <Route path="/tools/new" element={<ToolEditPage />} />
          <Route path="/tools/:id" element={<ToolEditPage />} />
          <Route path="/memories" element={<MemoriesPage />} />
          <Route path="/memories/new" element={<MemoryEditPage />} />
          <Route path="/memories/:id" element={<MemoryEditPage />} />
          <Route path="/templates" element={<TemplatesPage />} />
          <Route path="/templates/new" element={<TemplateEditPage />} />
          <Route path="/templates/:id" element={<TemplateEditPage />} />
          <Route path="/agents" element={<AgentsPage />} />
          <Route path="/agent-teams" element={<AgentTeamsPage />} />
          <Route path="/agent-teams/new" element={<AgentTeamEditPage />} />
          <Route path="/agent-teams/:id" element={<AgentTeamEditPage />} />
          <Route path="/instances" element={<InstancesPage />} />
          <Route path="/instances/:id/chat" element={<ChatPage />} />
          <Route path="/models" element={<ModelsPage />} />
          <Route path="/resources" element={<ResourcesPage />} />
        </Routes>
      </main>
    </div>
  )
}

export default App

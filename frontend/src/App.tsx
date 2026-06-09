import { Routes, Route, NavLink } from 'react-router-dom'
import PromptsPage from './pages/PromptsPage'
import PromptEditPage from './pages/PromptEditPage'
import SkillsPage from './pages/SkillsPage'
import SkillEditPage from './pages/SkillEditPage'
import ToolsPage from './pages/ToolsPage'
import ToolEditPage from './pages/ToolEditPage'
import TemplatesPage from './pages/TemplatesPage'
import TemplateEditPage from './pages/TemplateEditPage'
import AgentsPage from './pages/AgentsPage'
import InstancesPage from './pages/InstancesPage'

const navItems = [
  { to: '/prompts', label: 'Prompts' },
  { to: '/skills', label: 'Skills' },
  { to: '/tools', label: 'Tools' },
  { to: '/templates', label: 'Templates' },
  { to: '/agents', label: 'Agents' },
  { to: '/instances', label: 'Instances' },
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
          <Route path="/prompts" element={<PromptsPage />} />
          <Route path="/prompts/new" element={<PromptEditPage />} />
          <Route path="/prompts/:id" element={<PromptEditPage />} />
          <Route path="/skills" element={<SkillsPage />} />
          <Route path="/skills/new" element={<SkillEditPage />} />
          <Route path="/skills/:id" element={<SkillEditPage />} />
          <Route path="/tools" element={<ToolsPage />} />
          <Route path="/tools/new" element={<ToolEditPage />} />
          <Route path="/tools/:id" element={<ToolEditPage />} />
          <Route path="/templates" element={<TemplatesPage />} />
          <Route path="/templates/new" element={<TemplateEditPage />} />
          <Route path="/templates/:id" element={<TemplateEditPage />} />
          <Route path="/agents" element={<AgentsPage />} />
          <Route path="/instances" element={<InstancesPage />} />
        </Routes>
      </main>
    </div>
  )
}

export default App

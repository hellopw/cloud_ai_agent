import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import App from './App'
import AgentPage from './pages/AgentPage'

// Mock all page components to simplify testing and avoid API calls
vi.mock('./pages/PromptsPage', () => ({
  default: () => <div>PromptsPage</div>,
}))
vi.mock('./pages/PromptEditPage', () => ({
  default: () => <div>PromptEditPage</div>,
}))
vi.mock('./pages/SkillsPage', () => ({
  default: () => <div>SkillsPage</div>,
}))
vi.mock('./pages/SkillEditPage', () => ({
  default: () => <div>SkillEditPage</div>,
}))
vi.mock('./pages/ToolsPage', () => ({
  default: () => <div>ToolsPage</div>,
}))
vi.mock('./pages/ToolEditPage', () => ({
  default: () => <div>ToolEditPage</div>,
}))
vi.mock('./pages/TemplatesPage', () => ({
  default: () => <div>TemplatesPage</div>,
}))
vi.mock('./pages/TemplateEditPage', () => ({
  default: () => <div>TemplateEditPage</div>,
}))
vi.mock('./pages/AgentsPage', () => ({
  default: () => <div>AgentsPage</div>,
}))
vi.mock('./pages/InstancesPage', () => ({
  default: () => <div>InstancesPage</div>,
}))
vi.mock('./pages/ChatPage', () => ({
  default: () => <div>ChatPage</div>,
}))
vi.mock('./pages/ResourcesPage', () => ({
  default: () => <div>ResourcesPage</div>,
}))
vi.mock('./pages/ModelsPage', () => ({
  default: () => <div>ModelsPage</div>,
}))
vi.mock('./pages/AgentTeamsPage', () => ({
  default: () => <div>AgentTeamsPage</div>,
}))
vi.mock('./pages/AgentTeamEditPage', () => ({
  default: () => <div>AgentTeamEditPage</div>,
}))

describe('App routing', () => {
  it('renders Agent link in sidebar', () => {
    render(
      <MemoryRouter initialEntries={['/agent']}>
        <App />
      </MemoryRouter>
    )
    const links = screen.getAllByRole('link')
    const agentLink = links.find((link) => link.textContent === 'Agent')
    expect(agentLink).toBeDefined()
  })

  it('renders AgentPage at /agent route', () => {
    render(
      <MemoryRouter initialEntries={['/agent']}>
        <App />
      </MemoryRouter>
    )
    // Sidebar nav link for "Agent" is active, and the page renders mocked InstancesPage content
    expect(screen.getByText('InstancesPage')).toBeDefined()
  })

  it('renders PromptsPage at / route', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>
    )
    expect(screen.getByText('PromptsPage')).toBeDefined()
  })

  it('renders AgentTeamsPage at /agent-teams route', () => {
    render(
      <MemoryRouter initialEntries={['/agent-teams']}>
        <App />
      </MemoryRouter>
    )
    expect(screen.getByText('AgentTeamsPage')).toBeDefined()
  })

  it('renders InstancesPage at /instances route', () => {
    render(
      <MemoryRouter initialEntries={['/instances']}>
        <App />
      </MemoryRouter>
    )
    expect(screen.getByText('InstancesPage')).toBeDefined()
  })
})

describe('AgentPage', () => {
  function renderAgentPage(tab?: string) {
    const initialEntries = tab ? [`/agent?tab=${tab}`] : ['/agent']
    return render(
      <MemoryRouter initialEntries={initialEntries}>
        <AgentPage />
      </MemoryRouter>
    )
  }

  it('renders all four tab buttons', () => {
    renderAgentPage()
    expect(screen.getByText('Instances')).toBeDefined()
    expect(screen.getByText('Images')).toBeDefined()
    expect(screen.getByText('Teams')).toBeDefined()
    expect(screen.getByText('Templates')).toBeDefined()
  })

  it('defaults to Instances tab', () => {
    renderAgentPage()
    expect(screen.getByText('InstancesPage')).toBeDefined()
  })

  it('shows Images tab content when ?tab=images', () => {
    renderAgentPage('images')
    expect(screen.getByText('AgentsPage')).toBeDefined()
  })

  it('shows Teams tab content when ?tab=teams', () => {
    renderAgentPage('teams')
    expect(screen.getByText('AgentTeamsPage')).toBeDefined()
  })

  it('shows Templates tab content when ?tab=templates', () => {
    renderAgentPage('templates')
    expect(screen.getByText('TemplatesPage')).toBeDefined()
  })

  it('falls back to Instances tab for unknown tab param', () => {
    renderAgentPage('unknown')
    expect(screen.getByText('InstancesPage')).toBeDefined()
  })

  it('switches tab content on button click', () => {
    renderAgentPage()
    // Click the Teams tab button
    fireEvent.click(screen.getByText('Teams'))
    expect(screen.getByText('AgentTeamsPage')).toBeDefined()
  })
})

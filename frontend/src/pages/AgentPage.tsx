import { useSearchParams } from 'react-router-dom'
import InstancesPage from './InstancesPage'
import AgentsPage from './AgentsPage'
import AgentTeamsPage from './AgentTeamsPage'
import TemplatesPage from './TemplatesPage'

const tabs = [
  { key: 'instances', label: 'Instances', component: InstancesPage },
  { key: 'images', label: 'Images', component: AgentsPage },
  { key: 'teams', label: 'Teams', component: AgentTeamsPage },
  { key: 'templates', label: 'Templates', component: TemplatesPage },
]

export default function AgentPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const activeTab = searchParams.get('tab') || 'instances'

  const handleTabClick = (key: string) => {
    setSearchParams({ tab: key })
  }

  const ActiveComponent = tabs.find((t) => t.key === activeTab)?.component || InstancesPage

  return (
    <div>
      <div style={{
        display: 'flex',
        borderBottom: '1px solid var(--border)',
        marginBottom: 24,
        gap: 0,
      }}>
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => handleTabClick(tab.key)}
            style={{
              padding: '10px 20px',
              background: 'none',
              border: 'none',
              borderBottom: activeTab === tab.key ? '2px solid var(--accent)' : '2px solid transparent',
              color: activeTab === tab.key ? 'var(--accent)' : 'var(--text-muted)',
              fontSize: 14,
              fontWeight: activeTab === tab.key ? 600 : 400,
              cursor: 'pointer',
              transition: 'all 0.15s',
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <ActiveComponent key={activeTab} />
    </div>
  )
}

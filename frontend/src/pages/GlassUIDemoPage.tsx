import { useState } from 'react'

const stats = [
  { label: 'Active Agents', value: '2,847', sub: '+12.3% from last week' },
  { label: 'Requests / min', value: '34.2k', sub: 'Peak: 52.1k' },
  { label: 'Avg Latency', value: '87ms', sub: 'P99: 210ms' },
  { label: 'Uptime', value: '99.97%', sub: 'Last 30 days' },
]

const features = [
  { icon: '⚡', title: 'Real-time Streaming', desc: 'SSE-powered streaming with sub-100ms latency. Messages appear character by character with zero buffering.' },
  { icon: '🧠', title: 'Multi-Model Routing', desc: 'Intelligent request routing across Claude, GPT, and local models based on cost, speed, and capability.' },
  { icon: '🔒', title: 'Sandboxed Execution', desc: 'Every agent runs in an isolated Docker container with resource limits and network policies enforced.' },
  { icon: '📦', title: 'Template Marketplace', desc: 'Share and reuse agent templates. One-click deploy from a growing library of community-built configurations.' },
  { icon: '🔗', title: 'MCP Protocol', desc: 'Native Model Context Protocol support. Connect agents to tools, databases, and APIs through a standard interface.' },
  { icon: '📊', title: 'Observability Dashboard', desc: 'Trace every request end-to-end. Token usage, latency breakdowns, and cost attribution built in.' },
]

const listItems = [
  { icon: '📝', title: 'Blog Post Generator', sub: 'Template · 2.4k uses' },
  { icon: '🐛', title: 'Code Review Bot', sub: 'Template · 1.8k uses' },
  { icon: '📅', title: 'Meeting Scheduler', sub: 'Custom agent · Running' },
  { icon: '📧', title: 'Email Summarizer', sub: 'Custom agent · Stopped' },
]

export default function GlassUIDemoPage() {
  const [modalOpen, setModalOpen] = useState(false)
  const [toggles, setToggles] = useState({ notifications: true, analytics: false, autoScale: true })

  return (
    <div className="glass-demo">
      {/* ═══ header ═══ */}
      <div className="glass-demo-header">
        <h1>Glassmorphism Design System</h1>
        <p>#353a47 &times; #b8c4d1 &times; #e8eef5 &mdash; cool-tone frosted glass</p>
      </div>

      {/* ═══ stats ═══ */}
      <div className="glass-section-title">Overview</div>
      <div className="glass-grid">
        {stats.map((s) => (
          <div className="glass-panel" key={s.label}>
            <div className="glass-stat-label">{s.label}</div>
            <div className="glass-stat-value">{s.value}</div>
            <div className="glass-stat-sub">{s.sub}</div>
          </div>
        ))}
      </div>

      {/* ═══ buttons ═══ */}
      <div className="glass-section-title">Buttons</div>
      <div className="glass-panel" style={{ marginBottom: 36 }}>
        <div className="glass-btn-group">
          <button className="btn-glass">Primary Action</button>
          <button className="btn-glass-outline">Secondary</button>
          <button className="btn-glass-ghost">Ghost</button>
        </div>
        <div className="glass-btn-group" style={{ marginBottom: 0 }}>
          <button className="btn-glass" disabled style={{ opacity: 0.4, cursor: 'not-allowed' }}>Disabled</button>
          <button className="btn-glass" style={{ fontSize: 12, padding: '6px 14px' }}>Small</button>
          <button className="btn-glass" style={{ fontSize: 16, padding: '14px 32px' }}>Large</button>
        </div>
      </div>

      {/* ═══ features ═══ */}
      <div className="glass-section-title">Features</div>
      <div className="glass-grid-3">
        {features.map((f) => (
          <div className="glass-panel-light" key={f.title}>
            <div className="glass-feature-icon">{f.icon}</div>
            <div className="glass-feature-title">{f.title}</div>
            <div className="glass-feature-desc">{f.desc}</div>
          </div>
        ))}
      </div>

      {/* ═══ form + profile ═══ */}
      <div className="glass-section-title">Form Elements</div>
      <div className="glass-grid-2">
        <div className="glass-panel">
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 13, color: 'var(--text-muted)', marginBottom: 6 }}>Agent Name</label>
            <input className="glass-input" placeholder="e.g. Code Reviewer" />
          </div>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 13, color: 'var(--text-muted)', marginBottom: 6 }}>Model</label>
            <select className="glass-select" style={{ width: '100%' }}>
              <option>Claude Opus 4.6</option>
              <option>Claude Sonnet 4.6</option>
              <option>GPT-5</option>
            </select>
          </div>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 13, color: 'var(--text-muted)', marginBottom: 6 }}>System Prompt</label>
            <textarea className="glass-textarea" placeholder="You are a helpful assistant..." />
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 10, marginBottom: 16 }}>
            {(['notifications', 'analytics', 'autoScale'] as const).map((key) => (
              <label className="glass-toggle" key={key} onClick={() => setToggles((p) => ({ ...p, [key]: !p[key] }))}>
                <div className={`glass-toggle-track${toggles[key] ? ' on' : ''}`}>
                  <div className="glass-toggle-knob" />
                </div>
                {key === 'notifications' ? 'Notifications' : key === 'analytics' ? 'Analytics' : 'Auto-scale'}
              </label>
            ))}
          </div>

          <div style={{ display: 'flex', gap: 10 }}>
            <button className="btn-glass">Save Agent</button>
            <button className="btn-glass-outline">Cancel</button>
          </div>
        </div>

        {/* profile card */}
        <div className="glass-panel" style={{ textAlign: 'center' }}>
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <div className="glass-avatar">CD</div>
          </div>
          <h3 style={{ fontSize: 18, fontWeight: 600, marginBottom: 4 }}>Claude Dev</h3>
          <p style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 16 }}>Production Agent · v2.4.1</p>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'center', marginBottom: 20 }}>
            <span className="glass-pill glass-pill-dark">Claude Opus 4.6</span>
            <span className="glass-pill glass-pill-mid">High Priority</span>
            <span className="glass-pill glass-pill-light">Production</span>
          </div>
          <div className="glass-grid" style={{ gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 20 }}>
            <div style={{ textAlign: 'center' }}>
              <div style={{ fontSize: 22, fontWeight: 700, background: 'linear-gradient(135deg, var(--g1), var(--g3))', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>2.4k</div>
              <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Requests</div>
            </div>
            <div style={{ textAlign: 'center' }}>
              <div style={{ fontSize: 22, fontWeight: 700, background: 'linear-gradient(135deg, var(--g1), var(--g3))', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>98%</div>
              <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Success Rate</div>
            </div>
          </div>
          <button className="btn-glass-outline" onClick={() => setModalOpen(true)}>View Details</button>
        </div>
      </div>

      {/* ═══ list ═══ */}
      <div className="glass-section-title" style={{ marginTop: 36 }}>Agent List</div>
      <div className="glass-panel-light" style={{ marginBottom: 36 }}>
        {listItems.map((item) => (
          <div className="glass-list-item" key={item.title}>
            <div className="glass-list-item-left">
              <div className="glass-list-item-icon">{item.icon}</div>
              <div>
                <div className="glass-list-item-title">{item.title}</div>
                <div className="glass-list-item-sub">{item.sub}</div>
              </div>
            </div>
            <button className="btn-glass-outline" style={{ padding: '6px 14px', fontSize: 12 }}>Open</button>
          </div>
        ))}
      </div>

      {/* ═══ modal ═══ */}
      {modalOpen && (
        <div className="glass-modal-overlay" onClick={() => setModalOpen(false)}>
          <div className="glass-modal" onClick={(e) => e.stopPropagation()}>
            <h3>Agent Details</h3>
            <p>Claude Dev is configured with 8GB memory, 2 vCPUs, and has access to the GitHub and Slack integrations. Last deployed 3 hours ago.</p>
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
              <button className="btn-glass-ghost" onClick={() => setModalOpen(false)}>Close</button>
              <button className="btn-glass">Redeploy</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

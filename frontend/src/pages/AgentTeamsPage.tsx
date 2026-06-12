import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { agentTeamsApi, providerConfigsApi } from '../api/client'

const statusClass = (s: string) => {
  if (s === 'draft') return 'tag-draft'
  if (s === 'building') return 'tag-building'
  if (s === 'ready') return 'tag-ready'
  if (s === 'failed') return 'tag-failed'
  return ''
}

export default function AgentTeamsPage() {
  const [teams, setTeams] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [building, setBuilding] = useState('')
  const [viewLogId, setViewLogId] = useState<string | null>(null)
  const [logContent, setLogContent] = useState<string>('')
  const [showStartDialog, setShowStartDialog] = useState<string | null>(null)
  const [models, setModels] = useState<any[]>([])
  const [modelError, setModelError] = useState('')

  const load = async () => {
    try {
      const t = await agentTeamsApi.list()
      setTeams(t); setError('')
    } catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load(); providerConfigsApi.list().then(setModels).catch(() => {}) }, [])

  const handleBuild = async (id: string) => {
    setBuilding(id)
    try {
      await fetch('/api/agent-teams/' + id + '/build', { method: 'POST' })
      setTimeout(load, 2000)
    } catch (e: any) { setError(e.message) }
    finally { setBuilding('') }
  }

  const handleDelete = async (id: string, name: string) => {
    if (!confirm('Delete team "' + name + '"?')) return
    await agentTeamsApi.delete(id); load()
  }

  const handleViewLog = async (id: string) => {
    if (viewLogId === id) { setViewLogId(null); return }
    setViewLogId(id)
    try {
      const res = await fetch('/api/agent-teams/' + id + '/log')
      setLogContent(await res.text())
    } catch { setLogContent('Failed to load log') }
  }

  const handleStartTeam = async (teamId: string) => {
    try {
      await fetch('/api/agent-teams/' + teamId + '/start', { method: 'POST' })
      setShowStartDialog(null)
      window.location.hash = '#/instances'
      window.location.reload()
    } catch (e: any) { setModelError(e.message) }
  }

  if (loading) return <div className="content"><p>Loading...</p></div>

  return (
    <div>
      <div className="page-header">
        <h2>Agent Teams</h2>
        <Link to="/agent-teams/new" className="btn btn-primary">+ New Team</Link>
      </div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}

      {showStartDialog && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0,0,0,0.2)', backdropFilter: 'blur(6px)', WebkitBackdropFilter: 'blur(6px)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          zIndex: 1000,
        }} onClick={() => setShowStartDialog(null)}>
          <div className="card" style={{ width: 400, padding: 24 }} onClick={e => e.stopPropagation()}>
            <h3 style={{ marginBottom: 16 }}>Start Team</h3>
            {modelError && <p style={{ color: 'var(--danger)', fontSize: 13, marginBottom: 12 }}>{modelError}</p>}
            <p style={{ fontSize: 13, marginBottom: 16 }}>Start this team's container. All member agents will be initialized inside a single container.</p>
            <div className="form-actions" style={{ marginTop: 16 }}>
              <button onClick={() => handleStartTeam(showStartDialog)} className="btn btn-primary">
                Start Instance
              </button>
              <button onClick={() => setShowStartDialog(null)} className="btn btn-ghost">Cancel</button>
            </div>
          </div>
        </div>
      )}

      {teams.length === 0 ? (
        <div className="empty-state"><p>No agent teams yet.</p></div>
      ) : teams.map((t: any) => (
        <div key={t.id} className="card card-row">
          <div>
            <h3>{t.name} <span className={`tag ${statusClass(t.status)}`}>{t.status}</span></h3>
            <p>{t.repo_url} ({t.branch})</p>
            {t.error_msg && <p style={{ color: 'var(--danger)', fontSize: 13 }}>{t.error_msg}</p>}
            {t.image_tag && <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>Image: {t.image_tag}</p>}
            <p style={{ fontSize: 11, marginTop: 4 }}>
              Members: {(t.members || []).length} ({((t.members || []) as any[]).map((m: any) => `${m.name} (${m.role})`).join(', ')})
            </p>
            <p style={{ fontSize: 11, marginTop: 4 }}>Created: {new Date(t.created_at).toLocaleString()}</p>
          </div>
          <div className="card-actions">
            {(t.status === 'draft' || t.status === 'failed') && (
              <button onClick={() => handleBuild(t.id)} disabled={building === t.id} className="btn btn-primary">
                {building === t.id ? 'Building...' : 'Build'}
              </button>
            )}
            {(t.status === 'failed' || t.status === 'building') && (
              <button onClick={() => handleViewLog(t.id)} className="btn btn-ghost">
                {viewLogId === t.id ? 'Hide Log' : 'View Log'}
              </button>
            )}
            {t.status === 'ready' && (
              <button onClick={() => setShowStartDialog(t.id)} className="btn btn-primary">Start</button>
            )}
            <Link to={`/agent-teams/${t.id}`} className="btn btn-ghost">Edit</Link>
            <button onClick={() => handleDelete(t.id, t.name)} className="btn btn-danger">Delete</button>
          </div>
          {viewLogId === t.id && (
            <div style={{ marginTop: 12, width: '100%' }}>
              <h4 style={{ marginBottom: 8 }}>Build Log</h4>
              <pre style={{
                padding: 12, borderRadius: 6,
                maxHeight: 400, overflow: 'auto', fontSize: 12, whiteSpace: 'pre-wrap',
              }}>{logContent || 'Build log is empty or not found.'}</pre>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

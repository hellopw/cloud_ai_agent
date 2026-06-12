import { useState, useEffect } from 'react'
import { agentsApi, templatesApi, resourcesApi, providerConfigsApi } from '../api/client'

const statusClass = (s: string) => {
  if (s === 'draft') return 'tag-draft'
  if (s === 'building') return 'tag-building'
  if (s === 'ready') return 'tag-ready'
  if (s === 'failed') return 'tag-failed'
  return ''
}

const typeLabels: Record<string, string> = {
  git: 'Git Repo',
  database: 'Database',
  knowledge: 'Knowledge Base',
}

function parseConfig(config: string): Record<string, string> {
  try { return JSON.parse(config) } catch { return {} }
}

export default function AgentsPage() {
  const [agents, setAgents] = useState<any[]>([])
  const [templates, setTemplates] = useState<any[]>([])
  const [resources, setResources] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    name: '', template_id: '',
    repo_url: '', git_username: '', git_password: '', branch: 'main',
    resource_ids: [] as string[],
  })
  const [creating, setCreating] = useState(false)
  const [building, setBuilding] = useState('')
  const [editing, setEditing] = useState<string | null>(null)
  const [editForm, setEditForm] = useState({
    name: '', template_id: '',
    repo_url: '', git_username: '', git_password: '', branch: 'main',
    resource_ids: [] as string[],
  })
  const [saving, setSaving] = useState(false)
  const [expandedDf, setExpandedDf] = useState<string | null>(null)
  const [viewLogId, setViewLogId] = useState<string | null>(null)
  const [logContent, setLogContent] = useState<string>('')

  const load = async () => {
    try {
      const [a, t, r] = await Promise.all([
        agentsApi.list(), templatesApi.list(), resourcesApi.list()
      ])
      setAgents(a); setTemplates(t); setResources(r); setError('')
    } catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])

  const gitResources = resources.filter((r: any) => r.type === 'git')

  const onSelectGitResource = (resourceId: string, isEdit: boolean) => {
    const r = resources.find((r: any) => r.id === resourceId)
    if (!r) return
    const cfg = parseConfig(r.config)
    const patch = {
      repo_url: cfg.url || '',
      git_username: cfg.username || '',
      git_password: cfg.password || '',
      branch: cfg.branch || 'main',
    }
    if (isEdit) {
      setEditForm({ ...editForm, ...patch })
    } else {
      setForm({ ...form, ...patch })
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault(); setCreating(true); setError('')
    try {
      await agentsApi.create(form)
      setShowForm(false)
      setForm({
        name: '', template_id: '',
        repo_url: '', git_username: '', git_password: '', branch: 'main',
        resource_ids: [],
      })
      load()
    } catch (e: any) { setError(e.message) }
    finally { setCreating(false) }
  }

  const handleBuild = async (id: string) => {
    setBuilding(id)
    try {
      await fetch('/api/agents/' + id + '/build', { method: 'POST' })
      setTimeout(load, 2000)
    } catch (e: any) { setError(e.message) }
    finally { setBuilding('') }
  }

  const handleDelete = async (id: string, name: string) => {
    if (!confirm('Delete agent "' + name + '"?')) return
    await agentsApi.delete(id); load()
  }

  const handleEdit = (a: any) => {
    setEditing(a.id)
    setEditForm({
      name: a.name, template_id: a.template_id,
      repo_url: a.repo_url, git_username: a.git_username || '', git_password: a.git_password || '', branch: a.branch,
      resource_ids: a.resource_ids || [],
    })
  }

  const handleCopy = (a: any) => {
    setForm({
      name: a.name + ' (copy)',
      template_id: a.template_id,
      repo_url: a.repo_url,
      git_username: a.git_username || '',
      git_password: a.git_password || '',
      branch: a.branch,
      resource_ids: a.resource_ids || [],
    })
    setShowForm(true)
  }

  const handleViewLog = async (id: string) => {
    if (viewLogId === id) { setViewLogId(null); return }
    setViewLogId(id)
    try {
      const res = await fetch('/api/agents/' + id + '/log')
      setLogContent(await res.text())
    } catch { setLogContent('Failed to load log') }
  }

  const handleSaveEdit = async (e: React.FormEvent) => {
    e.preventDefault(); setSaving(true); setError('')
    try { await agentsApi.update(editing!, editForm); setEditing(null); load() }
    catch (err: any) { setError(err.message) }
    finally { setSaving(false) }
  }

  const getTemplate = (id: string) => templates.find((t: any) => t.id === id)
  const toggleDockerfile = (id: string) => setExpandedDf(expandedDf === id ? null : id)
  const getResourceName = (id: string) => resources.find((r: any) => r.id === id)?.name
  const [startModelId, setStartModelId] = useState('')
  const [showStartDialog, setShowStartDialog] = useState<string | null>(null)
  const [models, setModels] = useState<any[]>([])
  const [modelError, setModelError] = useState('')

  useEffect(() => {
    providerConfigsApi.list().then(setModels).catch(() => {})
  }, [])


  if (loading) return <div className="content"><p>Loading...</p></div>

  const handleStartInstance = async (agentId: string) => {
    if (!startModelId) { setModelError('Select a model first'); return }
    try {
      await fetch('/api/agents/' + agentId + '/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider_config_id: startModelId }),
      })
      setShowStartDialog(null)
      setStartModelId('')
      setModelError('')
      // Navigate to instances
      window.location.hash = '#/instances'
      window.location.reload()
    } catch (e: any) { setModelError(e.message) }
  }

  const StartButton = ({ agentId, agentName }: { agentId: string; agentName: string }) => (
    <>
      <button onClick={() => { setShowStartDialog(agentId); setStartModelId(''); setModelError('') }} className="btn btn-primary">
        Start
      </button>
      {showStartDialog === agentId && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0,0,0,0.2)', backdropFilter: 'blur(6px)', WebkitBackdropFilter: 'blur(6px)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          zIndex: 1000,
        }} onClick={() => setShowStartDialog(null)}>
          <div className="card" style={{ width: 400, padding: 24 }} onClick={e => e.stopPropagation()}>
            <h3 style={{ marginBottom: 16 }}>Start "{agentName}"</h3>
            {modelError && <p style={{ color: 'var(--danger)', fontSize: 13, marginBottom: 12 }}>{modelError}</p>}
            <div className="form-group">
              <label>Select Model Config</label>
              <select value={startModelId} onChange={e => setStartModelId(e.target.value)}>
                <option value="">-- Choose --</option>
                {models.map((m: any) => (
                  <option key={m.id} value={m.id}>{m.name} ({m.provider} / {m.model_id})</option>
                ))}
              </select>
            </div>
            <div className="form-actions" style={{ marginTop: 16 }}>
              <button onClick={() => handleStartInstance(agentId)} className="btn btn-primary" disabled={!startModelId}>
                Start Instance
              </button>
              <button onClick={() => setShowStartDialog(null)} className="btn btn-ghost">Cancel</button>
            </div>
          </div>
        </div>
      )}
    </>
  )

  const renderForm = (f: any, setF: any, onSubmit: any, submitLabel: string, loading: boolean, isEdit: boolean) => (
    <form className="form" onSubmit={onSubmit}>
      <div className="form-group">
        <label>Name</label>
        <input value={f.name} onChange={(e: any) => setF({ ...f, name: e.target.value })} required />
      </div>
      <div className="form-group">
        <label>Template</label>
        <select value={f.template_id} onChange={(e: any) => setF({ ...f, template_id: e.target.value })} required>
          <option value="">Select template...</option>
          {templates.map((t: any) => <option key={t.id} value={t.id}>{t.name} [{t.agent_type || 'pi'}]</option>)}
        </select>
      </div>

      <div className="form-group">
        <label>Git Resource (select existing or fill manually)</label>
        <select
          value=""
          onChange={(e: any) => { if (e.target.value) onSelectGitResource(e.target.value, isEdit) }}
        >
          <option value="">-- Pick a saved Git resource --</option>
          {gitResources.map((r: any) => (
            <option key={r.id} value={r.id}>{r.name}</option>
          ))}
        </select>
      </div>

      <div className="form-group">
        <label>Repository URL</label>
        <input value={f.repo_url} onChange={(e: any) => setF({ ...f, repo_url: e.target.value })} required placeholder="https://github.com/user/repo.git" />
      </div>
      <div className="form-group">
        <label>Git Username (optional)</label>
        <input value={f.git_username} onChange={(e: any) => setF({ ...f, git_username: e.target.value })} placeholder="username or oauth2" />
      </div>
      <div className="form-group">
        <label>Git Password / Token (optional)</label>
        <input type="password" value={f.git_password} onChange={(e: any) => setF({ ...f, git_password: e.target.value })} placeholder="personal access token" />
      </div>
      <div className="form-group">
        <label>Branch</label>
        <input value={f.branch} onChange={(e: any) => setF({ ...f, branch: e.target.value })} />
      </div>

      <div className="form-group">
        <label>Additional Resources (database, knowledge base, etc.)</label>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 6 }}>
          {resources.filter((r: any) => r.type !== 'git').map((r: any) => {
            const checked = f.resource_ids.includes(r.id)
            return (
              <label key={r.id} style={{
                display: 'flex', alignItems: 'center', gap: 4,
                padding: '4px 10px', border: '1px solid var(--border)', borderRadius: 4,
                background: checked ? 'var(--primary-light)' : 'transparent',
                cursor: 'pointer', fontSize: 13,
              }}>
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={() => {
                    setF({
                      ...f,
                      resource_ids: checked
                        ? f.resource_ids.filter((id: string) => id !== r.id)
                        : [...f.resource_ids, r.id],
                    })
                  }}
                />
                <span>{r.name}</span>
                <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>({typeLabels[r.type] || r.type})</span>
              </label>
            )
          })}
          {resources.filter((r: any) => r.type !== 'git').length === 0 && (
            <span style={{ color: 'var(--text-muted)', fontSize: 13 }}>No additional resources available. Add them in the Resources tab first.</span>
          )}
        </div>
      </div>

      <button type="submit" className="btn btn-primary" disabled={loading}>{loading ? submitLabel + '...' : submitLabel}</button>
    </form>
  )

  return (
    <div>
      <div className="page-header">
        <h2>Agents</h2>
        <button onClick={() => setShowForm(!showForm)} className="btn btn-primary">
          {showForm ? 'Cancel' : '+ New Agent'}
        </button>
      </div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}

      {showForm && (
        <div className="card" style={{ marginBottom: 24 }}>
          <h3 style={{ marginBottom: 16 }}>New Agent</h3>
          {renderForm(form, setForm, handleCreate, 'Creating', creating, false)}
        </div>
      )}

      {editing && (
        <div className="card" style={{ marginBottom: 24 }}>
          <h3 style={{ marginBottom: 16 }}>Edit Agent</h3>
          {renderForm(editForm, setEditForm, handleSaveEdit, 'Saving', saving, true)}
          <button type="button" onClick={() => setEditing(null)} className="btn btn-ghost" style={{ marginTop: 8 }}>Cancel</button>
        </div>
      )}

      {agents.length === 0 ? (
        <div className="empty-state"><p>No agents yet.</p></div>
      ) : agents.map((a: any) => (
        <div key={a.id} className="card card-row">
          <div>
            <h3>{a.name} <span className={`tag ${statusClass(a.status)}`}>{a.status}</span></h3>
            <p>{a.repo_url} ({a.branch})</p>
            {a.error_msg && <p style={{ color: 'var(--danger)', fontSize: 13 }}>{a.error_msg}</p>}
            {a.image_tag && <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>Image: {a.image_tag}</p>}
            <p style={{ fontSize: 11, marginTop: 4 }}>
              Template: {getTemplate(a.template_id)?.name || a.template_id}
            </p>
            {(a.resource_ids && a.resource_ids.length > 0) && (
              <p style={{ fontSize: 11, marginTop: 4 }}>
                Resources:{' '}
                {a.resource_ids.map((rid: string) => (
                  <span key={rid} className="tag tag-draft" style={{ marginRight: 4 }}>
                    {getResourceName(rid) || rid}
                  </span>
                ))}
              </p>
            )}
            <p style={{ fontSize: 11, marginTop: 4 }}>Created: {new Date(a.created_at).toLocaleString()}</p>
            <button
              onClick={() => toggleDockerfile(a.id)}
              style={{ marginTop: 8, background: 'none', border: '1px solid var(--border)', color: 'var(--text-muted)', borderRadius: 4, padding: '2px 10px', cursor: 'pointer', fontSize: 12 }}
            >
              {expandedDf === a.id ? 'Hide Dockerfile' : 'View Dockerfile'}
            </button>
            {expandedDf === a.id && (
              <pre style={{ marginTop: 8, maxHeight: 300, overflow: 'auto', fontSize: 12 }}>
                {getTemplate(a.template_id)?.dockerfile_content || '(empty)'}
              </pre>
            )}
          </div>
          <div className="card-actions">
            {(a.status === 'draft' || a.status === 'failed') && (
              <button onClick={() => handleBuild(a.id)} disabled={building === a.id} className="btn btn-primary">
                {building === a.id ? 'Building...' : 'Build'}
              </button>
            )}
            {(a.status === 'failed' || a.status === 'building') && (
              <button onClick={() => handleViewLog(a.id)} className="btn btn-ghost">
                {viewLogId === a.id ? 'Hide Log' : 'View Log'}
              </button>
            )}
            {a.status === 'ready' && (
              <StartButton agentId={a.id} agentName={a.name} />
            )}
            <button onClick={() => handleCopy(a)} className="btn btn-ghost">Copy</button>
            {(a.status === 'draft' || a.status === 'failed') && (
              <button onClick={() => handleEdit(a)} className="btn btn-ghost">Edit</button>
            )}
            <button onClick={() => handleDelete(a.id, a.name)} className="btn btn-danger">Delete</button>
          </div>
        </div>
      ))}
    </div>
  )
}

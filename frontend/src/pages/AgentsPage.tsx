import { useState, useEffect } from 'react'
import { agentsApi, templatesApi } from '../api/client'

const statusClass = (s: string) => {
  if (s === 'draft') return 'tag-draft'
  if (s === 'building') return 'tag-building'
  if (s === 'ready') return 'tag-ready'
  if (s === 'failed') return 'tag-failed'
  return ''
}

export default function AgentsPage() {
  const [agents, setAgents] = useState<any[]>([])
  const [templates, setTemplates] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ name: '', template_id: '', repo_url: '', git_username: '', git_password: '', branch: 'main' })
  const [creating, setCreating] = useState(false)
  const [building, setBuilding] = useState('')
  const [editing, setEditing] = useState<string | null>(null)
  const [editForm, setEditForm] = useState({ name: "", template_id: "", repo_url: "", git_username: "", git_password: "", branch: "main" })
  const [saving, setSaving] = useState(false)
  const [expandedDf, setExpandedDf] = useState<string | null>(null)
  const [viewLogId, setViewLogId] = useState<string | null>(null)
  const [logContent, setLogContent] = useState<string>("")

  const load = async () => {
    try {
      const [a, t] = await Promise.all([agentsApi.list(), templatesApi.list()])
      setAgents(a); setTemplates(t); setError('')
    } catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault(); setCreating(true); setError('')
    try { await agentsApi.create(form); setShowForm(false); load() }
    catch (e: any) { setError(e.message) }
    finally { setCreating(false) }
  }

  const handleBuild = async (id: string) => {
    setBuilding(id)
    try {
      await fetch(`/api/agents/${id}/build`, { method: 'POST' })
      setTimeout(load, 2000)
    } catch (e: any) { setError(e.message) }
    finally { setBuilding('') }
  }

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete agent "${name}"?`)) return; await agentsApi.delete(id); load()
  }
  const handleEdit = (a: any) => {
    setEditing(a.id)
    setEditForm({ name: a.name, template_id: a.template_id, repo_url: a.repo_url, git_username: a.git_username || '', git_password: a.git_password || '', branch: a.branch })
  }

  const handleCopy = (a: any) => {
    setForm({
      name: a.name + " (copy)",
      template_id: a.template_id,
      repo_url: a.repo_url,
      git_username: a.git_username || "",
      git_password: a.git_password || "",
      branch: a.branch
    })
    setShowForm(true)
  }

  const handleViewLog = async (id: string) => {
    if (viewLogId === id) { setViewLogId(null); return }
    setViewLogId(id)
    try {
      const res = await fetch('/api/agents/' + id + '/log')
      setLogContent(await res.text())
    } catch { setLogContent("Failed to load log") }
  }

  const handleSaveEdit = async (e: React.FormEvent) => {
    e.preventDefault(); setSaving(true); setError('')
    try { await agentsApi.update(editing!, editForm); setEditing(null); load() }
    catch (err: any) { setError(err.message) }
    finally { setSaving(false) }
  }

  const getTemplate = (id: string) => templates.find((t: any) => t.id === id)
  const toggleDockerfile = (id: string) => setExpandedDf(expandedDf === id ? null : id)

  if (loading) return <div className="content"><p>Loading...</p></div>

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
        <form className="form card" onSubmit={handleCreate} style={{ marginBottom: 24 }}>
          <div className="form-group">
            <label>Name</label>
            <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
          </div>
          <div className="form-group">
            <label>Template</label>
            <select value={form.template_id} onChange={(e) => setForm({ ...form, template_id: e.target.value })} required>
              <option value="">Select template...</option>
              {templates.map((t: any) => <option key={t.id} value={t.id}>{t.name}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label>Repository URL</label>
            <input value={form.repo_url} onChange={(e) => setForm({ ...form, repo_url: e.target.value })} required placeholder="https://github.com/user/repo.git" />
          </div>
          <div className="form-group">
            <label>Git Username (optional, for private repos)</label>
            <input value={form.git_username} onChange={(e) => setForm({ ...form, git_username: e.target.value })} placeholder="username or oauth2" />
          </div>
          <div className="form-group">
            <label>Git Password / Token (optional)</label>
            <input type="password" value={form.git_password} onChange={(e) => setForm({ ...form, git_password: e.target.value })} placeholder="personal access token" />
          </div>
          <div className="form-group">
            <label>Branch</label>
            <input value={form.branch} onChange={(e) => setForm({ ...form, branch: e.target.value })} />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>{creating ? 'Creating...' : 'Create Agent'}</button>
        </form>
      )}


      {editing && (
        <div className="card" style={{ marginTop: 16 }}>
          <h3 style={{ marginBottom: 16 }}>Edit Agent</h3>
          <form className="form" onSubmit={handleSaveEdit}>
            <div className="form-group"><label>Name</label><input value={editForm.name} onChange={(e) => setEditForm({ ...editForm, name: e.target.value })} required /></div>
            <div className="form-group">
              <label>Template</label>
              <select value={editForm.template_id} onChange={(e) => setEditForm({ ...editForm, template_id: e.target.value })} required>
                <option value="">Select template...</option>
                {templates.map((t: any) => <option key={t.id} value={t.id}>{t.name}</option>)}
              </select>
            </div>
            <div className="form-group"><label>Repository URL</label><input value={editForm.repo_url} onChange={(e) => setEditForm({ ...editForm, repo_url: e.target.value })} required /></div>
            <div className="form-group"><label>Git Username (optional)</label><input value={editForm.git_username} onChange={(e) => setEditForm({ ...editForm, git_username: e.target.value })} placeholder="username or oauth2" /></div>
            <div className="form-group"><label>Git Password / Token (optional)</label><input type="password" value={editForm.git_password} onChange={(e) => setEditForm({ ...editForm, git_password: e.target.value })} placeholder="leave empty to keep current" /></div>
            <div className="form-group"><label>Branch</label><input value={editForm.branch} onChange={(e) => setEditForm({ ...editForm, branch: e.target.value })} /></div>
            <div className="form-actions">
              <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? "Saving..." : "Save Changes"}</button>
              <button type="button" onClick={() => setEditing(null)} className="btn btn-ghost">Cancel</button>
            </div>
          </form>
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
            <p style={{ fontSize: 11, marginTop: 4 }}>Template: {getTemplate(a.template_id)?.name || a.template_id}</p>
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
            {(a.status === "failed" || a.status === "building") && (
              <button onClick={() => handleViewLog(a.id)} className="btn btn-ghost">
                {viewLogId === a.id ? "Hide Log" : "View Log"}
              </button>
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

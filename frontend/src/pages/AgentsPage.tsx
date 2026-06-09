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
  const [form, setForm] = useState({ name: '', template_id: '', repo_url: '', branch: 'main' })
  const [creating, setCreating] = useState(false)
  const [building, setBuilding] = useState('')

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
            <label>Branch</label>
            <input value={form.branch} onChange={(e) => setForm({ ...form, branch: e.target.value })} />
          </div>
          <button type="submit" className="btn btn-primary" disabled={creating}>{creating ? 'Creating...' : 'Create Agent'}</button>
        </form>
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
            <p style={{ fontSize: 11, marginTop: 4 }}>Created: {new Date(a.created_at).toLocaleString()}</p>
          </div>
          <div className="card-actions">
            {(a.status === 'draft' || a.status === 'failed') && (
              <button onClick={() => handleBuild(a.id)} disabled={building === a.id} className="btn btn-primary">
                {building === a.id ? 'Building...' : 'Build'}
              </button>
            )}
            <button onClick={() => handleDelete(a.id, a.name)} className="btn btn-danger">Delete</button>
          </div>
        </div>
      ))}
    </div>
  )
}

import { useState, useEffect } from 'react'
import { resourcesApi } from '../api/client'

const typeLabels: Record<string, string> = {
  git: 'Git Repo',
  database: 'Database',
  knowledge: 'Knowledge Base',
}

const typeFields: Record<string, { key: string; label: string; placeholder: string; password?: boolean }[]> = {
  git: [
    { key: 'url', label: 'Repository URL', placeholder: 'git@code.sohuno.com:user/repo.git' },
    { key: 'username', label: 'Username', placeholder: 'username or oauth2' },
    { key: 'password', label: 'Password / Token', placeholder: 'personal access token', password: true },
    { key: 'branch', label: 'Branch', placeholder: 'main' },
  ],
  database: [
    { key: 'db_type', label: 'Type', placeholder: 'mysql / postgresql / mongodb' },
    { key: 'connection', label: 'Connection', placeholder: 'host:port/database' },
    { key: 'username', label: 'Username', placeholder: 'db user' },
    { key: 'password', label: 'Password', placeholder: 'db password', password: true },
  ],
  knowledge: [
    { key: 'connection', label: 'API Endpoint', placeholder: 'https://kb.example.com/api' },
    { key: 'doc_url', label: 'Document URL', placeholder: 'https://docs.example.com' },
  ],
}

function parseConfig(config: string): Record<string, string> {
  try { return JSON.parse(config) } catch { return {} }
}

export default function ResourcesPage() {
  const [items, setItems] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ name: '', type: 'git', config: '{}' })
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [editing, setEditing] = useState<string | null>(null)
  const [editValues, setEditValues] = useState<Record<string, string>>({})
  const [editType, setEditType] = useState('git')

  const load = async () => {
    try {
      setItems(await resourcesApi.list())
      setError('')
    } catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault(); setSubmitting(true); setError('')
    try {
      await resourcesApi.create({
        name: form.name,
        type: form.type,
        config: JSON.stringify(fieldValues),
      })
      setShowForm(false)
      setForm({ name: '', type: 'git', config: '{}' })
      setFieldValues({})
      load()
    } catch (e: any) { setError(e.message) }
    finally { setSubmitting(false) }
  }

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete resource "${name}"?`)) return
    await resourcesApi.delete(id)
    load()
  }

  const handleEdit = (item: any) => {
    setEditing(item.id)
    setEditType(item.type)
    setEditValues(parseConfig(item.config))
  }

  const handleSaveEdit = async (e: React.FormEvent) => {
    e.preventDefault(); setSubmitting(true); setError('')
    try {
      const itemName = items.find((r: any) => r.id === editing)?.name || ''
      await resourcesApi.update(editing!, { name: itemName, type: editType, config: JSON.stringify(editValues) })
      setEditing(null)
      load()
    } catch (e: any) { setError(e.message) }
    finally { setSubmitting(false) }
  }

  if (loading) return <div className="content"><p>Loading...</p></div>

  const typeOptions = Object.keys(typeLabels)

  return (
    <div>
      <div className="page-header">
        <h2>Resources</h2>
        <button onClick={() => setShowForm(!showForm)} className="btn btn-primary">
          {showForm ? 'Cancel' : '+ New Resource'}
        </button>
      </div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}

      {showForm && (
        <form className="form card" onSubmit={handleCreate} style={{ marginBottom: 24 }}>
          <div className="form-group">
            <label>Name</label>
            <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required placeholder="My Git Repo" />
          </div>
          <div className="form-group">
            <label>Type</label>
            <select value={form.type} onChange={e => { setForm({ ...form, type: e.target.value }); setFieldValues({}) }}>
              {typeOptions.map(t => <option key={t} value={t}>{typeLabels[t]}</option>)}
            </select>
          </div>
          {typeFields[form.type].map(f => (
            <div className="form-group" key={f.key}>
              <label>{f.label}</label>
              <input
                type={f.password ? 'password' : 'text'}
                value={fieldValues[f.key] || ''}
                onChange={e => setFieldValues({ ...fieldValues, [f.key]: e.target.value })}
                placeholder={f.placeholder}
              />
            </div>
          ))}
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? 'Creating...' : 'Create Resource'}
          </button>
        </form>
      )}

      {editing && (
        <div className="card" style={{ marginBottom: 24 }}>
          <h3 style={{ marginBottom: 16 }}>Edit Resource</h3>
          <form className="form" onSubmit={handleSaveEdit}>
            <div className="form-group">
              <label>Type</label>
              <select value={editType} onChange={e => { setEditType(e.target.value); setEditValues({}) }}>
                {typeOptions.map(t => <option key={t} value={t}>{typeLabels[t]}</option>)}
              </select>
            </div>
            {typeFields[editType].map(f => (
              <div className="form-group" key={f.key}>
                <label>{f.label}</label>
                <input
                  type={f.password ? 'password' : 'text'}
                  value={editValues[f.key] || ''}
                  onChange={e => setEditValues({ ...editValues, [f.key]: e.target.value })}
                  placeholder={f.placeholder}
                />
              </div>
            ))}
            <div className="form-actions">
              <button type="submit" className="btn btn-primary" disabled={submitting}>{submitting ? 'Saving...' : 'Save'}</button>
              <button type="button" onClick={() => setEditing(null)} className="btn btn-ghost">Cancel</button>
            </div>
          </form>
        </div>
      )}

      {items.length === 0 ? (
        <div className="empty-state"><p>No resources yet.</p></div>
      ) : items.map((r: any) => (
        <div key={r.id} className="card card-row">
          <div>
            <h3>{r.name} <span className="tag tag-draft">{typeLabels[r.type] || r.type}</span></h3>
            {typeFields[r.type]?.map(f => {
              const val = parseConfig(r.config)[f.key]
              if (!val) return null
              return <p key={f.key} style={{fontSize:13,margin:0}}>{f.label}: {f.password ? '****' : val}</p>
            })}
            <p style={{ fontSize: 11, marginTop: 4, color: 'var(--text-muted)' }}>
              Created: {new Date(r.created_at).toLocaleString()}
            </p>
          </div>
          <div className="card-actions">
            <button onClick={() => handleEdit(r)} className="btn btn-ghost">Edit</button>
            <button onClick={() => handleDelete(r.id, r.name)} className="btn btn-danger">Delete</button>
          </div>
        </div>
      ))}
    </div>
  )
}
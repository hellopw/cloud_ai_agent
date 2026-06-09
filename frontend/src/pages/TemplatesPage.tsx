import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { templatesApi } from '../api/client'

export default function TemplatesPage() {
  const [items, setItems] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const load = async () => {
    try { setItems(await templatesApi.list()); setError('') }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])
  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete template "${name}"?`)) return; await templatesApi.delete(id); load()
  }
  if (loading) return <div className="content"><p>Loading...</p></div>
  return (
    <div>
      <div className="page-header"><h2>Templates</h2><Link to="/templates/new" className="btn btn-primary">+ New Template</Link></div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}
      {items.length === 0 ? (
        <div className="empty-state"><p>No templates yet.</p><Link to="/templates/new" className="btn btn-primary">Create Template</Link></div>
      ) : items.map((t: any) => (
        <div key={t.id} className="card card-row">
          <div>
            <h3>{t.name}</h3>
            <p>{t.description || 'No description'}</p>
            <p style={{ fontSize: 11, marginTop: 4 }}>Updated: {new Date(t.updated_at).toLocaleString()}</p>
          </div>
          <div className="card-actions">
            <Link to={`/templates/${t.id}`} className="btn btn-ghost">Edit</Link>
            <button onClick={() => handleDelete(t.id, t.name)} className="btn btn-danger">Delete</button>
          </div>
        </div>
      ))}
    </div>
  )
}

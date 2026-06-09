import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { toolsApi } from '../api/client'

export default function ToolsPage() {
  const [items, setItems] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const load = async () => {
    try { setItems(await toolsApi.list()); setError('') }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])
  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete tool "${name}"?`)) return; await toolsApi.delete(id); load()
  }
  if (loading) return <div className="content"><p>Loading...</p></div>
  return (
    <div>
      <div className="page-header"><h2>Tools</h2><Link to="/tools/new" className="btn btn-primary">+ New Tool</Link></div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}
      {items.length === 0 ? (
        <div className="empty-state"><p>No tools yet.</p><Link to="/tools/new" className="btn btn-primary">Create Tool</Link></div>
      ) : items.map((t: any) => (
        <div key={t.id} className="card card-row">
          <div>
            <h3>{t.label} <span style={{ color: 'var(--text-muted)', fontWeight: 400, fontSize: 13 }}>({t.name})</span></h3>
            <p>{t.description || 'No description'}</p>
            <p style={{ fontSize: 11, marginTop: 4 }}>Updated: {new Date(t.updated_at).toLocaleString()}</p>
          </div>
          <div className="card-actions">
            <Link to={`/tools/${t.id}`} className="btn btn-ghost">Edit</Link>
            <button onClick={() => handleDelete(t.id, t.name)} className="btn btn-danger">Delete</button>
          </div>
        </div>
      ))}
    </div>
  )
}

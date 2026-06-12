import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { memoriesApi } from '../api/client'

export default function MemoriesPage() {
  const [items, setItems] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const load = async () => {
    try { setItems(await memoriesApi.list()); setError('') }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])
  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete memory "${name}"?`)) return; await memoriesApi.delete(id); load()
  }
  if (loading) return <div className="content"><p>Loading...</p></div>
  return (
    <div>
      <div className="page-header"><h2>Memories</h2><Link to="/memories/new" className="btn btn-primary">+ New Memory</Link></div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}
      {items.length === 0 ? (
        <div className="empty-state"><p>No memories yet. Create one or run an agent to auto-extract.</p><Link to="/memories/new" className="btn btn-primary">Create Memory</Link></div>
      ) : items.map((m: any) => (
        <div key={m.id} className="card card-row">
          <div>
            <h3>{m.name}{m.source === 'auto' && <span className="tag tag-draft" style={{ marginLeft: 8 }}>auto</span>}</h3>
            <p>{m.description || 'No description'}</p>
            <p style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4, maxWidth: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {m.content?.substring(0, 120)}{(m.content?.length > 120) ? '...' : ''}
            </p>
            <p style={{ fontSize: 11, marginTop: 4 }}>Updated: {new Date(m.updated_at).toLocaleString()}</p>
          </div>
          <div className="card-actions">
            <Link to={`/memories/${m.id}`} className="btn btn-ghost">Edit</Link>
            <button onClick={() => handleDelete(m.id, m.name)} className="btn btn-danger">Delete</button>
          </div>
        </div>
      ))}
    </div>
  )
}

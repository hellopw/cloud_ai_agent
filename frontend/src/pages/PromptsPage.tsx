import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { promptsApi } from '../api/client'

interface Prompt {
  id: string
  name: string
  description: string
  updated_at: string
}

export default function PromptsPage() {
  const [items, setItems] = useState<Prompt[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    try { setItems(await promptsApi.list()); setError('') }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete prompt "${name}"?`)) return
    await promptsApi.delete(id)
    load()
  }

  if (loading) return <div className="content"><p>Loading...</p></div>

  return (
    <div>
      <div className="page-header">
        <h2>Prompts</h2>
        <Link to="/prompts/new" className="btn btn-primary">+ New Prompt</Link>
      </div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}
      {items.length === 0 ? (
        <div className="empty-state">
          <p>No prompts yet. Create your first prompt template.</p>
          <Link to="/prompts/new" className="btn btn-primary">Create Prompt</Link>
        </div>
      ) : items.map((p) => (
        <div key={p.id} className="card card-row">
          <div>
            <h3>{p.name}</h3>
            <p>{p.description || 'No description'}</p>
            <p style={{ fontSize: 11, marginTop: 4 }}>Updated: {new Date(p.updated_at).toLocaleString()}</p>
          </div>
          <div className="card-actions">
            <Link to={`/prompts/${p.id}`} className="btn btn-ghost">Edit</Link>
            <button onClick={() => handleDelete(p.id, p.name)} className="btn btn-danger">Delete</button>
          </div>
        </div>
      ))}
    </div>
  )
}

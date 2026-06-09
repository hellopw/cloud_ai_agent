import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { skillsApi } from '../api/client'

interface Skill {
  id: string; name: string; description: string; updated_at: string
}

export default function SkillsPage() {
  const [items, setItems] = useState<Skill[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    try { setItems(await skillsApi.list()); setError('') }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete skill "${name}"?`)) return
    await skillsApi.delete(id); load()
  }

  if (loading) return <div className="content"><p>Loading...</p></div>
  return (
    <div>
      <div className="page-header">
        <h2>Skills</h2>
        <Link to="/skills/new" className="btn btn-primary">+ New Skill</Link>
      </div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}
      {items.length === 0 ? (
        <div className="empty-state"><p>No skills yet.</p><Link to="/skills/new" className="btn btn-primary">Create Skill</Link></div>
      ) : items.map((s) => (
        <div key={s.id} className="card card-row">
          <div>
            <h3>{s.name}</h3>
            <p>{s.description || 'No description'}</p>
            <p style={{ fontSize: 11, marginTop: 4 }}>Updated: {new Date(s.updated_at).toLocaleString()}</p>
          </div>
          <div className="card-actions">
            <Link to={`/skills/${s.id}`} className="btn btn-ghost">Edit</Link>
            <button onClick={() => handleDelete(s.id, s.name)} className="btn btn-danger">Delete</button>
          </div>
        </div>
      ))}
    </div>
  )
}

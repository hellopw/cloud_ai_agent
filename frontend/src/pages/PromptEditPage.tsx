import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { promptsApi } from '../api/client'

export default function PromptEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isNew = !id

  const [form, setForm] = useState({ name: '', description: '', content: '' })
  const [loading, setLoading] = useState(!isNew)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (id) {
      promptsApi.get(id).then((p) => {
        setForm({ name: p.name, description: p.description, content: p.content })
        setLoading(false)
      }).catch((e) => { setError(e.message); setLoading(false) })
    }
  }, [id])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      if (isNew) {
        await promptsApi.create(form)
      } else {
        await promptsApi.update(id!, form)
      }
      navigate('/prompts')
    } catch (e: any) { setError(e.message) }
    finally { setSaving(false) }
  }

  if (loading) return <div className="content"><p>Loading...</p></div>

  return (
    <div>
      <div className="page-header">
        <h2>{isNew ? 'New Prompt' : 'Edit Prompt'}</h2>
      </div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}
      <form className="form" onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Name</label>
          <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
        </div>
        <div className="form-group">
          <label>Description</label>
          <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </div>
        <div className="form-group">
          <label>Content (Markdown)</label>
          <textarea value={form.content} onChange={(e) => setForm({ ...form, content: e.target.value })} required />
        </div>
        <div className="form-actions">
          <button type="submit" className="btn btn-primary" disabled={saving}>
            {saving ? 'Saving...' : 'Save'}
          </button>
          <Link to="/prompts" className="btn btn-ghost">Cancel</Link>
        </div>
      </form>
    </div>
  )
}

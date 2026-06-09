import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { templatesApi, promptsApi, skillsApi, toolsApi } from '../api/client'

export default function TemplateEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isNew = !id

  const [form, setForm] = useState({ name: '', description: '', dockerfile_content: '' })
  const [allPrompts, setAllPrompts] = useState<any[]>([])
  const [allSkills, setAllSkills] = useState<any[]>([])
  const [allTools, setAllTools] = useState<any[]>([])
  const [selectedPrompts, setSelectedPrompts] = useState<string[]>([])
  const [selectedSkills, setSelectedSkills] = useState<string[]>([])
  const [selectedTools, setSelectedTools] = useState<string[]>([])
  const [loading, setLoading] = useState(!isNew)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([promptsApi.list(), skillsApi.list(), toolsApi.list()]).then(([p, s, t]) => {
      setAllPrompts(p); setAllSkills(s); setAllTools(t)
      if (id) {
        templatesApi.get(id).then((tmpl: any) => {
          setForm({ name: tmpl.name, description: tmpl.description, dockerfile_content: tmpl.dockerfile_content || '' })
          setSelectedPrompts(tmpl.prompt_ids || [])
          setSelectedSkills(tmpl.skill_ids || [])
          setSelectedTools(tmpl.tool_ids || [])
          setLoading(false)
        })
      } else { setLoading(false) }
    }).catch((e) => { setError(e.message); setLoading(false) })
  }, [id])

  const toggle = (arr: string[], setArr: (a: string[]) => void, val: string) => {
    setArr(arr.includes(val) ? arr.filter((v) => v !== val) : [...arr, val])
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault(); setSaving(true); setError('')
    try {
      if (isNew) {
        const created = await templatesApi.create(form)
        await templatesApi.bind(created.id, { prompt_ids: selectedPrompts, skill_ids: selectedSkills, tool_ids: selectedTools })
      } else {
        await templatesApi.update(id!, form)
        await templatesApi.bind(id!, { prompt_ids: selectedPrompts, skill_ids: selectedSkills, tool_ids: selectedTools })
      }
      navigate('/templates')
    } catch (e: any) { setError(e.message) }
    finally { setSaving(false) }
  }

  if (loading) return <div className="content"><p>Loading...</p></div>

  return (
    <div>
      <div className="page-header"><h2>{isNew ? 'New Template' : 'Edit Template'}</h2></div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}
      <form className="form" onSubmit={handleSubmit}>
        <div className="form-group"><label>Name</label><input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
        <div className="form-group"><label>Description</label><input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></div>

        <div className="binding-section">
          <h4>Prompts ({selectedPrompts.length} selected)</h4>
          <div className="checkbox-list">
            {allPrompts.map((p: any) => (
              <label key={p.id} className={`checkbox-item ${selectedPrompts.includes(p.id) ? 'checked' : ''}`}>
                <input type="checkbox" checked={selectedPrompts.includes(p.id)} onChange={() => toggle(selectedPrompts, setSelectedPrompts, p.id)} style={{ display: 'none' }} />
                {p.name}
              </label>
            ))}
          </div>
        </div>

        <div className="binding-section">
          <h4>Skills ({selectedSkills.length} selected)</h4>
          <div className="checkbox-list">
            {allSkills.map((s: any) => (
              <label key={s.id} className={`checkbox-item ${selectedSkills.includes(s.id) ? 'checked' : ''}`}>
                <input type="checkbox" checked={selectedSkills.includes(s.id)} onChange={() => toggle(selectedSkills, setSelectedSkills, s.id)} style={{ display: 'none' }} />
                {s.name}
              </label>
            ))}
          </div>
        </div>

        <div className="binding-section">
          <h4>Tools ({selectedTools.length} selected)</h4>
          <div className="checkbox-list">
            {allTools.map((t: any) => (
              <label key={t.id} className={`checkbox-item ${selectedTools.includes(t.id) ? 'checked' : ''}`}>
                <input type="checkbox" checked={selectedTools.includes(t.id)} onChange={() => toggle(selectedTools, setSelectedTools, t.id)} style={{ display: 'none' }} />
                {t.label} ({t.name})
              </label>
            ))}
          </div>
        </div>

        <div className="form-group">
          <label>Dockerfile Content (optional — auto-generated if empty)</label>
          <textarea value={form.dockerfile_content} onChange={(e) => setForm({ ...form, dockerfile_content: e.target.value })} style={{ minHeight: 120, fontFamily: 'monospace', fontSize: 13 }} />
        </div>

        <div className="form-actions">
          <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
          <Link to="/templates" className="btn btn-ghost">Cancel</Link>
        </div>
      </form>
    </div>
  )
}

import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { toolsApi } from '../api/client'

const TEMPLATES: Record<string, string> = {
  http: JSON.stringify({
    name: "",
    label: "",
    description: "",
    parameters: {
      query: { type: "string", description: "Search query" }
    },
    handler: {
      type: "http",
      method: "GET",
      url: "https://api.example.com/search?q={{query}}",
      headers: { "Authorization": "Bearer {{env.API_KEY}}" }
    }
  }, null, 2),
  mcp_stdio: JSON.stringify({
    name: "",
    label: "",
    description: "",
    parameters: {
      input: { type: "string", description: "Input for the MCP tool" }
    },
    handler: {
      type: "mcp",
      transport: "stdio",
      command: "npx",
      args: ["-y", "@modelcontextprotocol/server-example"],
      env: {},
      tool_name: "example_tool"
    }
  }, null, 2),
  mcp_sse: JSON.stringify({
    name: "",
    label: "",
    description: "",
    parameters: {
      input: { type: "string", description: "Input for the MCP tool" }
    },
    handler: {
      type: "mcp",
      transport: "sse",
      url: "https://mcp-server.example.com/sse",
      tool_name: "example_tool"
    }
  }, null, 2),
  mcp_discover: JSON.stringify({
    name: "",
    label: "",
    description: "Discover available tools on an MCP server",
    parameters: {},
    handler: {
      type: "mcp",
      transport: "stdio",
      command: "npx",
      args: ["-y", "@modelcontextprotocol/server-example"],
      env: {}
    }
  }, null, 2),
  javascript: JSON.stringify({
    name: "",
    label: "",
    description: "",
    parameters: {
      input: { type: "string", description: "Input value" }
    },
    handler: {
      type: "javascript",
      code: "      return { content: [{ type: 'text', text: `Processed: ${args.input}` }] };"
    }
  }, null, 2),
}

export default function ToolEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isNew = !id
  const [form, setForm] = useState({ name: '', label: '', description: '', dsl_definition: '{\n  "handler": { "type": "http" }\n}' })
  const [loading, setLoading] = useState(!isNew)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (id) toolsApi.get(id).then((t: any) => {
      setForm({ name: t.name, label: t.label, description: t.description, dsl_definition: t.dsl_definition })
      setLoading(false)
    }).catch((e) => { setError(e.message); setLoading(false) })
  }, [id])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault(); setSaving(true); setError('')
    try {
      JSON.parse(form.dsl_definition)
      isNew ? await toolsApi.create(form) : await toolsApi.update(id!, form)
      navigate('/tools')
    } catch (e: any) { setError(e.message) }
    finally { setSaving(false) }
  }

  if (loading) return <div className="content"><p>Loading...</p></div>
  return (
    <div>
      <div className="page-header"><h2>{isNew ? 'New Tool' : 'Edit Tool'}</h2></div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}
      <form className="form" onSubmit={handleSubmit}>
        <div className="form-group"><label>Name</label><input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required placeholder="web_search" /></div>
        <div className="form-group"><label>Label</label><input value={form.label} onChange={(e) => setForm({ ...form, label: e.target.value })} required placeholder="Web Search" /></div>
        <div className="form-group"><label>Description</label><input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></div>
        <div className="form-group">
          <label>DSL Definition (JSON)</label>
          <div style={{ marginBottom: 8, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <span style={{ fontSize: 12, color: 'var(--text-muted)', lineHeight: '28px' }}>Templates:</span>
            {Object.entries(TEMPLATES).map(([key, value]) => (
              <button key={key} type="button" className="btn btn-sm btn-ghost"
                style={{ fontSize: 11, padding: '3px 8px' }}
                onClick={() => {
                  try {
                    // Pre-fill name/label from form if template includes them
                    const t = JSON.parse(value)
                    setForm(prev => ({
                      ...prev,
                      dsl_definition: JSON.stringify({ ...t, name: prev.name || t.name, label: prev.label || t.label }, null, 2)
                    }))
                  } catch { setForm({ ...form, dsl_definition: value }) }
                }}>
                {key.replace('_', ' ')}
              </button>
            ))}
          </div>
          <textarea
            value={form.dsl_definition}
            onChange={(e) => setForm({ ...form, dsl_definition: e.target.value })}
            style={{ minHeight: 300, fontFamily: 'monospace', fontSize: 13 }}
          />
        </div>
        <div className="form-actions">
          <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
          <Link to="/tools" className="btn btn-ghost">Cancel</Link>
        </div>
      </form>
    </div>
  )
}

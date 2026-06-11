import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { instancesApi, agentsApi, providerConfigsApi } from '../api/client'

const statusClass = (s: string) => {
  if (s === 'running') return 'tag-running'
  if (s === 'idle') return 'tag-ready'
  if (s === 'stopped') return 'tag-stopped'
  if (s === 'starting') return 'tag-building'
  return 'tag-failed'
}

const providerLabels: Record<string, string> = {
  "openai-codex": "OpenAI Codex",
  anthropic: "Anthropic (Claude)",
  openai: "OpenAI",
}

export default function InstancesPage() {
  const [items, setItems] = useState<any[]>([])
  const [agents, setAgents] = useState<any[]>([])
  const [models, setModels] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [starting, setStarting] = useState('')
  const [selectedModel, setSelectedModel] = useState<Record<string, string>>({})

  const load = async () => {
    try {
      const [inst, ag, md] = await Promise.all([instancesApi.list(), agentsApi.list(), providerConfigsApi.list()])
      setItems(inst); setAgents(ag); setModels(md); setError('')
    } catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])

  const handleStart = async (agentId: string) => {
    const modelId = selectedModel[agentId] || ''
    if (!modelId) { setError('Please select a model config first'); return }
    setStarting(agentId)
    try {
      await fetch(`/api/agents/${agentId}/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider_config_id: modelId }),
      })
      load()
    } catch (e: any) { setError(e.message) }
    finally { setStarting('') }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Stop and delete this instance?')) return
    await instancesApi.delete(id); load()
  }

  if (loading) return <div className="content"><p>Loading...</p></div>

  const runningInstances = items.filter((i) => i.status === 'running' || i.status === 'starting')
  const failedInstances = items.filter((i) => i.status === 'error' || i.status === 'stopped')
  const readyAgents = agents.filter((a: any) => a.status === 'ready')

  return (
    <div>
      <div className="page-header"><h2>Instances</h2></div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}

      <div className="card" style={{ marginBottom: 24 }}>
        <h3 style={{ marginBottom: 12 }}>Start New Instance</h3>
        {models.length === 0 ? (
          <p style={{ color: 'var(--text-muted)', fontSize: 13 }}>
            No model configs yet. Go to <b>Models</b> tab to create one first.
          </p>
        ) : readyAgents.length === 0 ? (
          <p style={{ color: 'var(--text-muted)', fontSize: 13 }}>No ready agents. Build an agent first.</p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {readyAgents.map((a: any) => (
              <div key={a.id} style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <span style={{ minWidth: 200, fontWeight: 600 }}>{a.name}</span>
                <select
                  value={selectedModel[a.id] || ''}
                  onChange={(e) => setSelectedModel({ ...selectedModel, [a.id]: e.target.value })}
                  style={{ flex: 1 }}
                >
                  <option value="">-- Select model --</option>
                  {models.map((m: any) => (
                    <option key={m.id} value={m.id}>
                      {m.name} ({providerLabels[m.provider] || m.provider} / {m.model_id})
                    </option>
                  ))}
                </select>
                <button
                  onClick={() => handleStart(a.id)}
                  disabled={starting === a.id || !selectedModel[a.id]}
                  className="btn btn-primary"
                >
                  {starting === a.id ? 'Starting...' : 'Start'}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {runningInstances.length === 0 && failedInstances.length === 0 ? (
        <div className="empty-state"><p>No instances yet.</p></div>
      ) : (
        <>
          {runningInstances.length > 0 && (
            <h3 style={{ margin: '24px 0 12px', fontSize: 14 }}>Running</h3>
          )}
          {runningInstances.map((i: any) => (
        <div key={i.id} className="card card-row">
          <div>
            <h3>
              Instance {i.id.substring(0, 8)}...{' '}
              <span className={`tag ${statusClass(i.status)}`}>{i.status}</span>
            </h3>
            <p>Agent: {i.agent_id} | Port: {i.host_port || 'N/A'}</p>
            <p style={{ fontSize: 11, marginTop: 4 }}>Started: {new Date(i.created_at).toLocaleString()}</p>
          </div>
          <div className="card-actions">
            {i.status === 'running' && (
              <Link to={`/instances/${i.id}/chat`} className="btn btn-primary">Chat</Link>
            )}
            <button onClick={() => handleDelete(i.id)} className="btn btn-danger">Stop</button>
          </div>
        </div>
      ))}
          {failedInstances.length > 0 && (
            <>
              <h3 style={{ margin: '24px 0 12px', fontSize: 14, color: 'var(--text-muted)' }}>Failed / Stopped</h3>
              {failedInstances.map((i: any) => (
                <div key={i.id} className="card card-row" style={{ opacity: 0.7 }}>
                  <div>
                    <h3>
                      Instance {i.id.substring(0, 8)}...{' '}
                      <span className={`tag ${statusClass(i.status)}`}>{i.status}</span>
                    </h3>
                    <p style={{ fontSize: 11 }}>Created: {new Date(i.created_at).toLocaleString()}</p>
                    {i.error_msg && (
                      <p style={{ color: 'var(--danger)', fontSize: 12, marginTop: 4 }}>{i.error_msg}</p>
                    )}
                  </div>
                  <div className="card-actions">
                    <button onClick={() => handleDelete(i.id)} className="btn btn-ghost">Remove</button>
                  </div>
                </div>
              ))}
            </>
          )}
        </>
      )}
    </div>
  )
}

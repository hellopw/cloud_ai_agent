import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { instancesApi, agentsApi } from '../api/client'

const statusClass = (s: string) => {
  if (s === 'running') return 'tag-running'
  if (s === 'idle') return 'tag-ready'
  if (s === 'stopped') return 'tag-stopped'
  if (s === 'starting') return 'tag-building'
  return 'tag-failed'
}

export default function InstancesPage() {
  const [items, setItems] = useState<any[]>([])
  const [agents, setAgents] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [starting, setStarting] = useState('')

  const load = async () => {
    try {
      const [inst, ag] = await Promise.all([instancesApi.list(), agentsApi.list()])
      setItems(inst); setAgents(ag); setError('')
    } catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])

  const handleStart = async (agentId: string) => {
    setStarting(agentId)
    try {
      await fetch(`/api/agents/${agentId}/start`, { method: 'POST' })
      load()
    } catch (e: any) { setError(e.message) }
    finally { setStarting('') }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Stop and delete this instance?')) return
    await instancesApi.delete(id); load()
  }

  if (loading) return <div className="content"><p>Loading...</p></div>

  const runningInstances = items.filter((i) => i.status !== 'stopped' && i.status !== 'error')

  return (
    <div>
      <div className="page-header"><h2>Instances</h2></div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}

      <div className="card" style={{ marginBottom: 24 }}>
        <h3 style={{ marginBottom: 12 }}>Start New Instance</h3>
        {agents.filter((a: any) => a.status === 'ready').length === 0 ? (
          <p style={{ color: 'var(--text-muted)', fontSize: 13 }}>No ready agents. Build an agent first.</p>
        ) : (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {agents.filter((a: any) => a.status === 'ready').map((a: any) => (
              <button
                key={a.id}
                onClick={() => handleStart(a.id)}
                disabled={starting === a.id}
                className="btn btn-primary"
                style={{ fontSize: 12 }}
              >
                {starting === a.id ? 'Starting...' : `Start ${a.name}`}
              </button>
            ))}
          </div>
        )}
      </div>

      {runningInstances.length === 0 ? (
        <div className="empty-state"><p>No running instances.</p></div>
      ) : runningInstances.map((i: any) => (
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
    </div>
  )
}

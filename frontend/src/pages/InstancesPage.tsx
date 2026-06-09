import { useState, useEffect } from 'react'
import { instancesApi } from '../api/client'

const statusClass = (s: string) => {
  if (s === 'running') return 'tag-running'
  if (s === 'idle') return 'tag-ready'
  if (s === 'stopped') return 'tag-stopped'
  if (s === 'starting') return 'tag-building'
  return 'tag-failed'
}

export default function InstancesPage() {
  const [items, setItems] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const load = async () => {
    try { setItems(await instancesApi.list()); setError('') }
    catch (e: any) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])

  const handleDelete = async (id: string) => {
    if (!confirm('Stop and delete this instance?')) return; await instancesApi.delete(id); load()
  }

  if (loading) return <div className="content"><p>Loading...</p></div>
  return (
    <div>
      <div className="page-header"><h2>Instances</h2></div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}
      {items.length === 0 ? (
        <div className="empty-state"><p>No running instances.</p></div>
      ) : items.map((i: any) => (
        <div key={i.id} className="card card-row">
          <div>
            <h3>Instance {i.id.substring(0, 8)}... <span className={`tag ${statusClass(i.status)}`}>{i.status}</span></h3>
            <p>Agent: {i.agent_id} | Port: {i.host_port || 'N/A'} | Container: {i.container_id || 'N/A'}</p>
            <p style={{ fontSize: 11, marginTop: 4 }}>Started: {new Date(i.created_at).toLocaleString()}</p>
          </div>
          <div className="card-actions">
            <button onClick={() => handleDelete(i.id)} className="btn btn-danger">Stop</button>
          </div>
        </div>
      ))}
    </div>
  )
}

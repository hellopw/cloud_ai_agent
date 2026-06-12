import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { instancesApi, fetchText } from '../api/client'

interface LogFile {
  name: string
  size: number
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function groupFiles(files: LogFile[]) {
  const snapshots: LogFile[] = []   // tools.json, prompts.json, skills.json
  const requests: LogFile[] = []    // request-N.jsonl
  const responses: LogFile[] = []   // response-N.jsonl
  const others: LogFile[] = []

  for (const f of files) {
    const base = f.name.includes('/') ? f.name.split('/').pop()! : f.name
    if (base === 'tools.json' || base === 'prompts.json' || base === 'skills.json') {
      snapshots.push(f)
    } else if (base.startsWith('request-')) {
      requests.push(f)
    } else if (base.startsWith('response-')) {
      responses.push(f)
    } else {
      others.push(f)
    }
  }

  return { snapshots, requests, responses, others }
}

export default function InstanceLogsPage() {
  const { id } = useParams<{ id: string }>()
  const [files, setFiles] = useState<LogFile[]>([])
  const [selected, setSelected] = useState<string>('')
  const [content, setContent] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [contentLoading, setContentLoading] = useState(false)
  const [error, setError] = useState('')

  // Load file list
  useEffect(() => {
    if (!id) return
    setLoading(true)
    instancesApi.llmLogs(id)
      .then((data: any) => {
        const sorted = (data.files || []).sort((a: LogFile, b: LogFile) =>
          a.name.localeCompare(b.name)
        )
        setFiles(sorted)
        if (sorted.length > 0) {
          // Auto-select tools.json or first file
          const toolsFile = sorted.find((f: LogFile) =>
            f.name.endsWith('/tools.json') || f.name === 'tools.json'
          )
          setSelected(toolsFile?.name || sorted[0].name)
        }
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))
  }, [id])

  // Load file content when selection changes
  useEffect(() => {
    if (!id || !selected) return
    setContentLoading(true)
    setContent('')
    fetchText(`/instances/${id}/llm-logs/${selected}`)
      .then(setContent)
      .catch((e: Error) => setError(e.message))
      .finally(() => setContentLoading(false))
  }, [id, selected])

  const { snapshots, requests, responses, others } = groupFiles(files)

  const renderContent = () => {
    if (contentLoading) {
      return <div className="empty-state"><p>Loading...</p></div>
    }
    if (!content) {
      return <div className="empty-state"><p>No content</p></div>
    }

    const isJsonl = selected.endsWith('.jsonl')
    const isJson = selected.endsWith('.json')

    if (isJsonl) {
      const lines = content.trim().split('\n').filter(Boolean)
      return (
        <div style={{ padding: 8 }}>
          {lines.map((line, i) => {
            let obj: any = null
            try { obj = JSON.parse(line) } catch { /* not json */ }
            return (
              <div
                key={i}
                style={{
                  marginBottom: 4,
                  border: '1px solid var(--border)',
                  borderRadius: 4,
                  overflow: 'hidden',
                }}
              >
                {obj ? (
                  <>
                    <div
                      style={{
                        padding: '4px 10px',
                        fontSize: 11,
                        background: 'var(--card-bg)',
                        color: obj.type === 'error' || obj.type === 'fatal_error'
                          ? 'var(--danger)'
                          : obj.type === 'tool_use'
                          ? 'var(--accent)'
                          : 'var(--text-muted)',
                        borderBottom: '1px solid var(--border)',
                        display: 'flex',
                        justifyContent: 'space-between',
                      }}
                    >
                      <span>
                        {obj.type && <strong>{obj.type}</strong>}
                        {obj.type === 'tool_use' && obj.name && ` · ${obj.name}`}
                        {obj.type === 'text_delta' && ' · text'}
                      </span>
                      {obj.ts && (
                        <span>{new Date(obj.ts).toLocaleTimeString()}</span>
                      )}
                    </div>
                    <pre style={{
                      margin: 0,
                      padding: '8px 10px',
                      fontSize: 12,
                      lineHeight: 1.5,
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-all',
                      background: '#0d1117',
                      color: '#c9d1d9',
                      maxHeight: obj.type === 'text_delta' ? 'none' : 600,
                      overflow: 'auto',
                    }}>
                      {JSON.stringify(obj, null, 2)}
                    </pre>
                  </>
                ) : (
                  <pre style={{
                    margin: 0,
                    padding: '8px 10px',
                    fontSize: 12,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-all',
                    color: 'var(--text)',
                  }}>
                    {line}
                  </pre>
                )}
              </div>
            )
          })}
        </div>
      )
    }

    if (isJson) {
      let obj: any
      try { obj = JSON.parse(content) } catch { return <pre style={preStyle}>{content}</pre> }
      return (
        <pre style={preStyle}>
          {JSON.stringify(obj, null, 2)}
        </pre>
      )
    }

    return <pre style={preStyle}>{content}</pre>
  }

  const renderFileList = (label: string, fileList: LogFile[]) => {
    if (fileList.length === 0) return null
    return (
      <div style={{ marginBottom: 16 }}>
        <div style={{
          fontSize: 11,
          color: 'var(--text-muted)',
          textTransform: 'uppercase',
          letterSpacing: 0.5,
          marginBottom: 4,
          paddingLeft: 4,
        }}>
          {label}
        </div>
        {fileList.map((f) => (
          <button
            key={f.name}
            onClick={() => setSelected(f.name)}
            style={{
              display: 'block',
              width: '100%',
              textAlign: 'left',
              padding: '6px 8px',
              fontSize: 13,
              border: 'none',
              borderRadius: 4,
              background: selected === f.name ? 'var(--accent)' : 'transparent',
              color: selected === f.name ? '#fff' : 'var(--text)',
              cursor: 'pointer',
              marginBottom: 2,
            }}
          >
            <span style={{ display: 'block' }}>{f.name}</span>
            <span style={{
              fontSize: 10,
              color: selected === f.name ? 'rgba(255,255,255,0.7)' : 'var(--text-muted)',
            }}>
              {formatSize(f.size)}
            </span>
          </button>
        ))}
      </div>
    )
  }

  return (
    <div>
      <div className="page-header" style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <Link to="/instances" style={{ color: 'var(--text-muted)', textDecoration: 'none', fontSize: 14 }}>
          &larr; Instances
        </Link>
        <h2>LLM Logs</h2>
        <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
          Instance {id?.substring(0, 8)}...
        </span>
      </div>

      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}

      {loading ? (
        <div className="empty-state"><p>Loading...</p></div>
      ) : files.length === 0 ? (
        <div className="card" style={{ padding: 32, textAlign: 'center' }}>
          <p style={{ color: 'var(--text-muted)', marginBottom: 8 }}>No LLM log files yet.</p>
          <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>
            Log files are created when the agent makes LLM API calls. Start a chat session first.
          </p>
        </div>
      ) : (
        <div style={{ display: 'flex', gap: 16, alignItems: 'flex-start' }}>
          {/* File list sidebar */}
          <div style={{
            width: 220,
            flexShrink: 0,
            background: 'var(--card-bg)',
            borderRadius: 'var(--radius)',
            padding: 12,
            maxHeight: 'calc(100vh - 120px)',
            overflow: 'auto',
            position: 'sticky',
            top: 16,
          }}>
            {renderFileList('Snapshots', snapshots)}
            {renderFileList('Requests', requests)}
            {renderFileList('Responses', responses)}
            {renderFileList('Other', others)}
          </div>

          {/* Content area */}
          <div style={{
            flex: 1,
            background: 'var(--card-bg)',
            borderRadius: 'var(--radius)',
            minHeight: 400,
            maxHeight: 'calc(100vh - 120px)',
            overflow: 'auto',
          }}>
            <div style={{
              position: 'sticky',
              top: 0,
              zIndex: 1,
              background: 'var(--card-bg)',
              borderBottom: '1px solid var(--border)',
              padding: '8px 16px',
              fontSize: 13,
              color: 'var(--text-muted)',
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}>
              <span><strong>{selected}</strong></span>
              {contentLoading && <span>Loading...</span>}
            </div>
            {renderContent()}
          </div>
        </div>
      )}
    </div>
  )
}

const preStyle: React.CSSProperties = {
  margin: 0,
  padding: 16,
  fontSize: 12,
  lineHeight: 1.5,
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
  color: '#c9d1d9',
  background: '#0d1117',
}

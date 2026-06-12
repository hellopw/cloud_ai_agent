import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'

interface Message {
  role: 'user' | 'assistant' | 'tool' | 'system'
  content: string
  toolCall?: { toolCallId: string; toolName: string; input: any }
}

interface ConfigItem {
  id: string
  name: string
  description?: string
  content?: string
  label?: string
  dsl_definition?: string
}

interface InstanceConfig {
  prompts: ConfigItem[]
  skills: ConfigItem[]
  tools: ConfigItem[]
}

export default function ChatPage() {
  const { id } = useParams<{ id: string }>()
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [streamingContent, setStreamingContent] = useState('')
  const [containerInfo, setContainerInfo] = useState<{ provider?: string; model?: string }>({})
  const [instanceInfo, setInstanceInfo] = useState<{ host_port: number; status: string }>({ host_port: 0, status: '' })
  const [instanceConfig, setInstanceConfig] = useState<InstanceConfig>({ prompts: [], skills: [], tools: [] })
  const [configOpen, setConfigOpen] = useState(false)
  const streamingRef = useRef('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const rafRef = useRef<number>(0)
  const needsFlushRef = useRef(false)
  const idRef = useRef(id)
  const wasThinkingRef = useRef(false)

  // Keep idRef in sync so closures always have the correct instance ID
  idRef.current = id

  // Batch update streaming content — at most once per animation frame
  const flushStreaming = () => {
    needsFlushRef.current = false
    setStreamingContent(streamingRef.current)
  }

  const scheduleContentUpdate = () => {
    if (!needsFlushRef.current) {
      needsFlushRef.current = true
      rafRef.current = requestAnimationFrame(flushStreaming)
    }
  }

  const cancelPendingFlush = () => {
    if (rafRef.current) {
      cancelAnimationFrame(rafRef.current)
      rafRef.current = 0
    }
    needsFlushRef.current = false
  }

  // Persist a chat message to the backend
  const saveMessage = (role: string, content: string, toolCall?: any) => {
    const instanceId = idRef.current
    if (!instanceId || !content) {
      console.warn('[ChatPage] saveMessage skipped', { instanceId, role, hasContent: !!content })
      return
    }
    const url = '/api/instances/' + instanceId + '/messages'
    console.log('[ChatPage] saveMessage', { url, role, contentLen: content.length })
    fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role, content, tool_call: toolCall ? JSON.stringify(toolCall) : '' }),
    }).then(r => {
      if (!r.ok) console.error('[ChatPage] saveMessage failed', r.status, r.statusText)
    }).catch(e => console.error('[ChatPage] saveMessage error', e))
  }

  // Load persisted messages on mount
  useEffect(() => {
    if (!id) return
    const url = '/api/instances/' + id + '/messages'
    console.log('[ChatPage] loading messages from', url)
    fetch(url)
      .then(r => r.json())
      .then((data: any[]) => {
        console.log('[ChatPage] loaded messages', data.length)
        if (Array.isArray(data) && data.length > 0) {
          const msgs: Message[] = data.map((m: any) => ({
            role: m.role,
            content: m.content,
            toolCall: m.tool_call ? JSON.parse(m.tool_call) : undefined,
          }))
          setMessages(msgs)
        }
      })
      .catch(() => {})
  }, [id])

  useEffect(() => {
    // Fetch instance info
    fetch('/api/instances/' + id)
      .then(r => r.json())
      .then(data => setInstanceInfo(data))
      .catch(() => {})
  }, [id])

  // Fetch associated prompts, skills, tools
  useEffect(() => {
    if (!id) return
    fetch('/api/instances/' + id + '/config')
      .then(r => r.json())
      .then(data => setInstanceConfig({
        prompts: data.prompts || [],
        skills: data.skills || [],
        tools: data.tools || [],
      }))
      .catch(() => {})
  }, [id])


  // Fetch container status via backend proxy
  useEffect(() => {
    if (id) {
      fetch('/api/instances/' + id + '/status')
        .then(r => r.json())
        .then(data => setContainerInfo(data))
        .catch(() => {})
    }
  }, [id])

  // Cleanup on unmount or id change
  useEffect(() => {
    return () => {
      cancelPendingFlush()
    }
  }, [id])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingContent])

  // SSE fallback for when WebSocket is blocked by corporate proxy
  const sendViaHttp = async (trimmed: string) => {
    const instanceId = idRef.current
    if (!instanceId) {
      console.error('[ChatPage] sendViaHttp: no instance id')
      setMessages((prev) => [...prev, { role: 'system', content: 'Error: no instance id' }])
      return
    }
    console.log('[ChatPage] sendViaHttp: sending to', instanceId)
    try {
      const resp = await fetch('/api/instances/' + instanceId + '/chat-http', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: trimmed }),
      })
      if (!resp.ok) {
        const text = await resp.text()
        throw new Error('HTTP ' + resp.status + ': ' + text)
      }
      const reader = resp.body?.getReader()
      if (!reader) throw new Error('No response body')

      const decoder = new TextDecoder()
      let buffer = ''
      let lastEventType = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const rawLines = buffer.split('\n')
        buffer = rawLines.pop() || ''
        for (const rawLine of rawLines) {
          const line = rawLine.trim()
          if (!line) continue
          try {
            if (line.startsWith('event: ')) {
              lastEventType = line.substring(7)
              continue
            }
            if (line.startsWith('data: ')) {
              const dataStr = line.substring(6)
              const parsed = JSON.parse(dataStr)
              processEvent({ type: lastEventType || 'text_delta', data: parsed })
            }
          } catch {
            // Skip malformed lines
          }
        }
      }
      // Process remaining buffer
      if (buffer.trim()) {
        try {
          if (buffer.startsWith('data: ')) {
            const parsed = JSON.parse(buffer.substring(6))
            processEvent({ type: 'agent_end', data: parsed })
          }
        } catch { /* ignore */ }
      }
      // Signal end of streaming
      if (streamingRef.current) {
        setMessages((msgs) => [...msgs, { role: 'assistant', content: streamingRef.current }])
        saveMessage('assistant', streamingRef.current)
      }
      cancelPendingFlush()
      streamingRef.current = ''
      setStreamingContent('')
    } catch (e: unknown) {
      console.error('[ChatPage] HTTP fallback error', e)
      setMessages((prev) => [...prev, {
        role: 'system',
        content: 'Error: ' + (e instanceof Error ? e.message : String(e)),
      }])
    }
  }

  // Shared event processing for both WebSocket and HTTP fallback
  const processEvent = (eventData: { type: string; data: any }) => {
    switch (eventData.type) {
      case 'text_delta':
        streamingRef.current += (eventData.data.delta || '')
        scheduleContentUpdate()
        break
      case 'tool_call':
        setMessages((prev) => [...prev, {
          role: 'tool',
          content: 'Calling: ' + eventData.data.toolName,
          toolCall: {
            toolCallId: eventData.data.toolCallId,
            toolName: eventData.data.toolName,
            input: eventData.data.input,
          },
        }])
        saveMessage('tool', 'Calling: ' + eventData.data.toolName, { toolCallId: eventData.data.toolCallId, toolName: eventData.data.toolName, input: eventData.data.input })
        break
      case 'tool_result':
        setMessages((prev) => [...prev, {
          role: 'system',
          content: 'Result from tool: ' + JSON.stringify(eventData.data.content).substring(0, 200),
        }])
        saveMessage('system', 'Result from tool: ' + JSON.stringify(eventData.data.content).substring(0, 200))
        break
      case 'message_update':
        const me3 = eventData.data?.assistantMessageEvent
        if (!me3) break
        switch (me3.type) {
          case 'thinking_delta':
            wasThinkingRef.current = true
            if (!streamingRef.current.startsWith('[Thinking]\n')) {
              streamingRef.current = '[Thinking]\n' + streamingRef.current
            }
            streamingRef.current += (me3.delta || '')
            scheduleContentUpdate()
            break
          case 'text_delta':
            if (wasThinkingRef.current) {
              streamingRef.current += '\n\n'
              wasThinkingRef.current = false
            }
            streamingRef.current += (me3.delta || '')
            scheduleContentUpdate()
            break
        }
        break
      case 'agent_end':
        console.log('[ChatPage] processEvent agent_end, streamingLen=' + streamingRef.current.length)
        if (streamingRef.current) {
          setMessages((msgs) => [...msgs, { role: 'assistant', content: streamingRef.current }])
          saveMessage('assistant', streamingRef.current)
        }
        cancelPendingFlush()
        streamingRef.current = ''
        wasThinkingRef.current = false
        setStreamingContent('')
        break
      case 'message_end':
        const me4 = eventData.data
        if (me4?.message?.stopReason === 'error') {
          setMessages((prev) => [...prev, { role: 'system', content: 'Error: ' + (me4.message.errorMessage || 'Unknown error') }])
        } else if (streamingRef.current) {
          setMessages((msgs) => [...msgs, { role: 'assistant', content: streamingRef.current }])
          saveMessage('assistant', streamingRef.current)
          cancelPendingFlush()
          streamingRef.current = ''
          wasThinkingRef.current = false
          setStreamingContent('')
        }
        break
      case 'error':
        if (streamingRef.current) {
          setMessages((msgs) => [...msgs, { role: 'assistant', content: streamingRef.current }])
          saveMessage('assistant', streamingRef.current)
          cancelPendingFlush()
          streamingRef.current = ''
          wasThinkingRef.current = false
          setStreamingContent('')
        }
        setMessages((prev) => [...prev, { role: 'system', content: 'Error: ' + eventData.data.message }])
        break
      default:
        setMessages((prev) => [...prev, { role: 'system', content: `[${eventData.type}] ${JSON.stringify(eventData.data).substring(0, 300)}` }])
    }
  }

  const handleSend = () => {
    const trimmed = input.trim()
    if (!trimmed) return
    setMessages((prev) => [...prev, { role: 'user', content: trimmed }])
    saveMessage('user', trimmed)
    setInput('')
    sendViaHttp(trimmed)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 64px)' }}>
      <div style={{
        padding: '12px 20px',
        borderBottom: '1px solid var(--border)',
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        background: 'var(--sidebar-bg)',
      }}>
        <Link to="/instances" className="btn btn-ghost" style={{ fontSize: 12 }}>Back</Link>
        <h3 style={{ fontSize: 15, margin: 0 }}>Instance {id?.substring(0, 8)}...</h3>
        <span className="tag tag-ready">connected</span>
        {containerInfo.provider && (
          <>
            <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>|</span>
            <span className="tag tag-draft">{containerInfo.provider}</span>
            {containerInfo.model && (
              <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{containerInfo.model}</span>
            )}
          </>
        )}
        {(instanceConfig.prompts.length > 0 || instanceConfig.skills.length > 0 || instanceConfig.tools.length > 0) && (
          <>
            <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>|</span>
            <button
              onClick={() => setConfigOpen(!configOpen)}
              className="btn btn-ghost"
              style={{ fontSize: 12, padding: '2px 8px' }}
            >
              Prompts({instanceConfig.prompts.length}) Skills({instanceConfig.skills.length}) Tools({instanceConfig.tools.length})
            </button>
          </>
        )}
      </div>

      {configOpen && (
        <div style={{
          padding: '16px 20px',
          borderBottom: '1px solid var(--border)',
          background: 'var(--bg)',
          display: 'flex',
          gap: 24,
          overflow: 'auto',
        }}>
          {instanceConfig.prompts.length > 0 && (
            <div style={{ minWidth: 200 }}>
              <h4 style={{ fontSize: 13, margin: '0 0 6px 0' }}>Prompts</h4>
              {instanceConfig.prompts.map(p => (
                <div key={p.id} style={{ fontSize: 12, marginBottom: 4 }}>
                  <strong>{p.name}</strong>
                  {p.description && <div style={{ color: 'var(--text-muted)', whiteSpace: 'pre-wrap', wordBreak: 'break-word', maxHeight: 100, overflow: 'auto' }}>{p.description}</div>}
                </div>
              ))}
            </div>
          )}
          {instanceConfig.skills.length > 0 && (
            <div style={{ minWidth: 200 }}>
              <h4 style={{ fontSize: 13, margin: '0 0 6px 0' }}>Skills</h4>
              {instanceConfig.skills.map(s => (
                <div key={s.id} style={{ fontSize: 12, marginBottom: 4 }}>
                  <strong>{s.name}</strong>
                  {s.description && <div style={{ color: 'var(--text-muted)', whiteSpace: 'pre-wrap', wordBreak: 'break-word', maxHeight: 100, overflow: 'auto' }}>{s.description}</div>}
                </div>
              ))}
            </div>
          )}
          {instanceConfig.tools.length > 0 && (
            <div style={{ minWidth: 200 }}>
              <h4 style={{ fontSize: 13, margin: '0 0 6px 0' }}>Tools</h4>
              {instanceConfig.tools.map(t => (
                <div key={t.id} style={{ fontSize: 12, marginBottom: 4 }}>
                  <strong>{t.label || t.name}</strong>
                  {t.description && <div style={{ color: 'var(--text-muted)', whiteSpace: 'pre-wrap', wordBreak: 'break-word', maxHeight: 100, overflow: 'auto' }}>{t.description}</div>}
                </div>
              ))}
            </div>
          )}
          {instanceConfig.prompts.length === 0 && instanceConfig.skills.length === 0 && instanceConfig.tools.length === 0 && (
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>No prompts, skills, or tools configured for this instance.</span>
          )}
        </div>
      )}

      <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
        {messages.map((msg, i) => (
          <div key={i} style={{
            marginBottom: 16,
            display: 'flex',
            justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
          }}>
            <div style={{
              maxWidth: '80%',
              padding: '10px 16px',
              borderRadius: 'var(--radius)',
              background: msg.role === 'user' ? 'var(--accent)' :
                msg.role === 'tool' ? 'rgba(240,192,64,0.1)' :
                msg.role === 'system' ? 'var(--bg)' : 'var(--card-bg)',
              color: msg.role === 'user' ? '#fff' : 'var(--text)',
              border: msg.role !== 'user' ? '1px solid var(--border)' : 'none',
              fontSize: 13,
              lineHeight: 1.6,
            }}>
              {msg.toolCall && (
                <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>
                  Tool: {msg.toolCall.toolName}
                </div>
              )}
              <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{msg.content}</div>
            </div>
          </div>
        ))}

        {streamingContent && (
          <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-start' }}>
            <div style={{
              maxWidth: '80%',
              padding: '10px 16px',
              borderRadius: 'var(--radius)',
              background: 'var(--card-bg)',
              border: '1px solid var(--border)',
              fontSize: 13,
              lineHeight: 1.6,
            }}>
              <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                {streamingContent}
                <span className="cursor-blink" style={{
                  display: 'inline-block',
                  width: 8,
                  height: 14,
                  background: 'var(--accent)',
                  marginLeft: 2,
                  animation: 'blink 1s step-end infinite',
                  verticalAlign: 'middle',
                }} />
              </div>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      <div style={{
        padding: '16px 24px',
        borderTop: '1px solid var(--border)',
        background: 'var(--sidebar-bg)',
      }}>
        <div style={{ display: 'flex', gap: 10 }}>
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a message... (Enter to send, Shift+Enter for new line)"
            rows={2}
            style={{
              flex: 1,
              padding: '10px 14px',
              background: 'var(--bg)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius)',
              color: 'var(--text)',
              fontSize: 14,
              fontFamily: 'inherit',
              resize: 'none',
            }}
          />
          <button onClick={handleSend} className="btn btn-primary" style={{ alignSelf: 'flex-end' }}>
            Send
          </button>
        </div>
      </div>

      <style>{`
        @keyframes blink {
          50% { opacity: 0; }
        }
      `}</style>
    </div>
  )
}

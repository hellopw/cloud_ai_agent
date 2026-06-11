import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'

interface Message {
  role: 'user' | 'assistant' | 'tool' | 'system'
  content: string
  toolCall?: { toolCallId: string; toolName: string; input: any }
}

export default function ChatPage() {
  const { id } = useParams<{ id: string }>()
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [connected, setConnected] = useState(false)
  const [streamingContent, setStreamingContent] = useState('')
  const [containerInfo, setContainerInfo] = useState<{ provider?: string; model?: string }>({})
  const [instanceInfo, setInstanceInfo] = useState<{ host_port: number; status: string }>({ host_port: 0, status: '' })
  const wsRef = useRef<WebSocket | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const streamingRef = useRef('')

  useEffect(() => {
    // Fetch instance info
    fetch('/api/instances/' + id)
      .then(r => r.json())
      .then(data => setInstanceInfo(data))
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

  useEffect(() => {
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null
    let ws: WebSocket | null = null

    const connect = () => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const url = `${protocol}//${window.location.host}/api/instances/${id}/chat`
      console.log('[ChatPage] WebSocket connecting to', url)
      ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        console.log('[ChatPage] WebSocket connected')
        setConnected(true)
      }
      ws.onclose = (e) => {
        console.log('[ChatPage] WebSocket closed, code=' + e.code + ' reason=' + e.reason)
        setConnected(false)
        // Auto-reconnect after 3 seconds
        reconnectTimer = setTimeout(connect, 3000)
      }
      ws.onerror = (e) => {
        console.error('[ChatPage] WebSocket error', e)
      }

      ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)

      switch (msg.type) {
        case 'text_delta':
          streamingRef.current += (msg.data.delta || '')
          setStreamingContent(streamingRef.current)
          break
        case 'tool_call':
          const tc = msg.data
          setMessages((prev) => [...prev, {
            role: 'tool',
            content: `Calling: ${tc.toolName}`,
            toolCall: { toolCallId: tc.toolCallId, toolName: tc.toolName, input: tc.input },
          }])
          break
        case 'tool_result':
          setMessages((prev) => [...prev, {
            role: 'system',
            content: `Result from tool: ${JSON.stringify(msg.data.content).substring(0, 200)}`,
          }])
          break
        case 'agent_end':
          if (streamingRef.current) {
            setMessages((msgs) => [...msgs, { role: 'assistant', content: streamingRef.current }])
          }
          streamingRef.current = ''
          setStreamingContent('')
          break
        case 'error':
          setMessages((prev) => [...prev, { role: 'system', content: `Error: ${msg.data.message}` }])
          break
      }
    }

    }
    connect()
    return () => {
      if (reconnectTimer) clearTimeout(reconnectTimer)
      if (ws) ws.close()
      wsRef.current = null
    }
  }, [id])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingContent])

  // SSE fallback for when WebSocket is blocked by corporate proxy
  const sendViaHttp = async (trimmed: string) => {
    console.log('[ChatPage] HTTP fallback: sending message')
    try {
      const resp = await fetch('/api/instances/' + id + '/chat-http', {
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
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        // Process complete SSE lines
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''
        for (const rawLine of lines) {
          const line = rawLine.trim()
          if (!line) continue
          try {
            // SSE format: "event: TYPE" or "data: {...}"  
            if (line.startsWith('event: ')) {
              // Currently events come as "event: TYPE\ndata: {...}" pairs,
              // but they arrive on separate lines from the reader.
              // We handle them in the next iteration when data line arrives.
              continue
            }
            if (line.startsWith('data: ')) {
              const dataStr = line.substring(6)
              // Check previous line for event type
              const prevLine = lines[lines.length - 1]?.trim() || ''
              let eventType = ''
              if (prevLine.startsWith('event: ')) {
                eventType = prevLine.substring(7)
              }
              const parsed = JSON.parse(dataStr)
              processEvent({ type: eventType || 'text_delta', data: parsed })
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
      }
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
        setStreamingContent(streamingRef.current)
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
        break
      case 'tool_result':
        setMessages((prev) => [...prev, {
          role: 'system',
          content: 'Result from tool: ' + JSON.stringify(eventData.data.content).substring(0, 200),
        }])
        break
      case 'agent_end':
        if (streamingRef.current) {
          setMessages((msgs) => [...msgs, { role: 'assistant', content: streamingRef.current }])
        }
        streamingRef.current = ''
        setStreamingContent('')
        break
      case 'error':
        setMessages((prev) => [...prev, { role: 'system', content: 'Error: ' + eventData.data.message }])
        break
    }
  }

  const handleSend = () => {
    const trimmed = input.trim()
    console.log('[ChatPage] handleSend called', { input, trimmed, connected, wsReadyState: wsRef.current?.readyState })
    if (!trimmed) {
      console.log('[ChatPage] handleSend: empty input, returning')
      return
    }
    const wsState = wsRef.current?.readyState
    if (wsState !== WebSocket.OPEN) {
      // WebSocket not available, fall back to HTTP SSE
      console.log('[ChatPage] handleSend: WebSocket not OPEN (state=' + wsState + '), using HTTP fallback')
      setMessages((prev) => [...prev, { role: 'user', content: trimmed }])
      setInput('')
      sendViaHttp(trimmed)
      return
    }
    console.log('[ChatPage] handleSend: sending via WebSocket')
    setMessages((prev) => [...prev, { role: 'user', content: trimmed }])
    try {
      wsRef.current!.send(JSON.stringify({ type: 'chat', message: trimmed }))
      setInput('')
    } catch (e) {
      console.error('[ChatPage] handleSend: send failed, falling back to HTTP', e)
      setMessages((prev) => [...prev, { role: 'user', content: trimmed }])
      setInput('')
      sendViaHttp(trimmed)
    }
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
        <span className={`tag ${connected ? 'tag-ready' : 'tag-failed'}`} title={`WebSocket readyState: ${wsRef.current?.readyState}`}>
          {connected ? 'connected' : 'disconnected [' + (wsRef.current?.readyState ?? 'null') + ']'}
        </span>
        {containerInfo.provider && (
          <>
            <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>|</span>
            <span className="tag tag-draft">{containerInfo.provider}</span>
            {containerInfo.model && (
              <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{containerInfo.model}</span>
            )}
          </>
        )}
      </div>

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
            placeholder={connected ? "Type a message... (Enter to send, Shift+Enter for new line)" : "Disconnected \u2014 start the instance to chat"}
            disabled={!connected}
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
          <button onClick={handleSend} disabled={!connected} className="btn btn-primary" style={{ alignSelf: 'flex-end', opacity: connected ? 1 : 0.5 }}>
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

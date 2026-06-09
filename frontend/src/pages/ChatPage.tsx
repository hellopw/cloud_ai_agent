import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { instancesApi } from '../api/client'

interface WSEvent {
  type: string
  data: any
}

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
  const wsRef = useRef<WebSocket | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/instances/${id}/chat`)

    ws.onopen = () => setConnected(true)
    ws.onclose = () => setConnected(false)

    ws.onmessage = (event) => {
      const msg: WSEvent = JSON.parse(event.data)

      switch (msg.type) {
        case 'text_delta':
          setStreamingContent((prev) => prev + (msg.data.delta || ''))
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
          setStreamingContent((prev) => {
            if (prev) {
              setMessages((msgs) => [...msgs, { role: 'assistant', content: prev }])
            }
            return ''
          })
          break
        case 'error':
          setMessages((prev) => [...prev, { role: 'system', content: `Error: ${msg.data.message}` }])
          break
      }
    }

    wsRef.current = ws
    return () => ws.close()
  }, [id])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingContent])

  const handleSend = () => {
    if (!input.trim() || !wsRef.current) return
    setMessages((prev) => [...prev, { role: 'user', content: input }])
    wsRef.current.send(JSON.stringify({ type: 'chat', message: input }))
    setInput('')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <div style={{
        padding: '12px 20px',
        borderBottom: '1px solid var(--border)',
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        background: 'var(--sidebar-bg)',
      }}>
        <Link to="/instances" className="btn btn-ghost" style={{ fontSize: 12 }}>Back</Link>
        <h3 style={{ fontSize: 15, margin: 0 }}>Chat with Instance {id?.substring(0, 8)}...</h3>
        <span className={`tag ${connected ? 'tag-ready' : 'tag-failed'}`}>
          {connected ? 'connected' : 'disconnected'}
        </span>
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

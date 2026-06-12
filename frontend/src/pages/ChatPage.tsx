import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'

type ContentBlock =
  | { type: 'text'; text: string }
  | { type: 'thinking'; thinking: string }
  | { type: 'tool_use'; toolCallId: string; toolName: string; input: any; result?: string }

interface Message {
  role: 'user' | 'assistant' | 'system'
  content: ContentBlock[]
}

/** Return a fresh copy of the blocks array with the last block replaced. */
function replaceLast(blocks: ContentBlock[], updated: ContentBlock): ContentBlock[] {
  const copy = [...blocks]
  copy[copy.length - 1] = updated
  return copy
}

export default function ChatPage() {
  const { id } = useParams<{ id: string }>()
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [connected, setConnected] = useState(false)
  const [streamingBlocks, setStreamingBlocks] = useState<ContentBlock[]>([])
  const [containerInfo, setContainerInfo] = useState<{ provider?: string; model?: string }>({})
  const [instanceInfo, setInstanceInfo] = useState<{ host_port: number; status: string }>({ host_port: 0, status: '' })
  const wsRef = useRef<WebSocket | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const streamingBlocksRef = useRef<ContentBlock[]>([])
  const messageCommittedRef = useRef(false)
  const idRef = useRef(id)

  // Keep idRef in sync so closures always have the correct instance ID
  idRef.current = id

  // Persist a chat message to the backend
  const saveMessage = (role: string, content: ContentBlock[]) => {
    const instanceId = idRef.current
    if (!instanceId || content.length === 0) return
    fetch('/api/instances/' + instanceId + '/messages', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role, content: JSON.stringify(content) }),
    }).then(r => {
      if (!r.ok) console.error('[ChatPage] saveMessage failed', r.status)
    }).catch(e => console.error('[ChatPage] saveMessage error', e))
  }

  // Load persisted messages on mount
  useEffect(() => {
    if (!id) return
    fetch('/api/instances/' + id + '/messages')
      .then(r => r.json())
      .then((data: any[]) => {
        if (Array.isArray(data) && data.length > 0) {
          const msgs: Message[] = data.map((m: any) => ({
            role: m.role,
            content: typeof m.content === 'string' ? JSON.parse(m.content)
              : Array.isArray(m.content) ? m.content
              : [{ type: 'text', text: String(m.content || '') }],
          }))
          setMessages(msgs)
        }
      })
      .catch(() => {})
  }, [id])

  useEffect(() => {
    fetch('/api/instances/' + id)
      .then(r => r.json())
      .then(data => setInstanceInfo(data))
      .catch(() => {})
  }, [id])

  useEffect(() => {
    if (id) {
      fetch('/api/instances/' + id + '/status')
        .then(r => r.json())
        .then(data => setContainerInfo(data))
        .catch(() => {})
    }
  }, [id])

  // ── shared event processor (WebSocket + HTTP SSE fallback) ──

  const processEvent = (eventType: string, data: any) => {
    console.log('[ChatPage] event:', eventType, JSON.stringify(data).substring(0, 100))
    switch (eventType) {
      case 'tool_call': {
        const toolCallId = data.toolCallId || data.id || `tc_${Date.now()}`
        const toolName = data.toolName || data.name || 'unknown'
        const input = data.input || data.arguments || {}
        streamingBlocksRef.current = [
          ...streamingBlocksRef.current,
          { type: 'tool_use', toolCallId, toolName, input },
        ]
        setStreamingBlocks([...streamingBlocksRef.current])
        break
      }

      case 'tool_result': {
        const resultId = data.toolCallId || data.tool_use_id || data.id || ''
        const resultContent =
          typeof data.content === 'string'
            ? data.content
            : JSON.stringify(data.content, null, 2)
        const blocks = streamingBlocksRef.current

        // Find matching tool_use block (last unmatched, prefer exact ID match)
        let matchIdx = -1
        for (let i = blocks.length - 1; i >= 0; i--) {
          const b = blocks[i]
          if (b.type !== 'tool_use') continue
          if (b.result !== undefined) continue
          if (resultId && b.toolCallId === resultId) {
            matchIdx = i
            break // exact ID match wins immediately
          }
          if (matchIdx === -1) {
            matchIdx = i // fallback: last unmatched tool_use
          }
        }

        if (matchIdx !== -1) {
          const block = blocks[matchIdx] as ContentBlock & { type: 'tool_use' }
          const updated: ContentBlock & { type: 'tool_use' } = { ...block, result: resultContent }
          const copy = [...blocks]
          copy[matchIdx] = updated
          streamingBlocksRef.current = copy
          setStreamingBlocks([...copy])
        }
        break
      }

      case 'message_update': {
        const me = data?.assistantMessageEvent
        if (!me) break

        // If the event carries a "partial" with structured content blocks,
        // use it directly — this is the complete message state from the provider.
        if (me.partial?.content && Array.isArray(me.partial.content)) {
          const blocks: ContentBlock[] = []
          for (const item of me.partial.content) {
            if (item.type === 'thinking') {
              blocks.push({ type: 'thinking', thinking: item.thinking || '' })
            } else if (item.type === 'text') {
              blocks.push({ type: 'text', text: item.text || '' })
            } else if (item.type === 'tool_use') {
              blocks.push({
                type: 'tool_use',
                toolCallId: item.id || String(blocks.length),
                toolName: item.name || 'unknown',
                input: item.input || {},
              })
            }
          }
          streamingBlocksRef.current = blocks
          setStreamingBlocks([...blocks])
          break
        }

        // Fallback: delta-by-delta accumulation
        switch (me.type) {
          case 'thinking_delta': {
            const delta = me.delta || ''
            if (!delta) break
            const blocks = streamingBlocksRef.current
            const last = blocks[blocks.length - 1]
            if (last && last.type === 'thinking') {
              streamingBlocksRef.current = replaceLast(blocks, { ...last, thinking: last.thinking + delta })
            } else {
              streamingBlocksRef.current = [...blocks, { type: 'thinking', thinking: delta }]
            }
            setStreamingBlocks([...streamingBlocksRef.current])
            break
          }
          case 'text_delta': {
            const delta = me.delta || ''
            if (!delta) break
            const blocks = streamingBlocksRef.current
            const last = blocks[blocks.length - 1]
            if (last && last.type === 'text') {
              streamingBlocksRef.current = replaceLast(blocks, { ...last, text: last.text + delta })
            } else {
              streamingBlocksRef.current = [...blocks, { type: 'text', text: delta }]
            }
            setStreamingBlocks([...streamingBlocksRef.current])
            break
          }
        }
        break
      }

      case 'agent_end': {
        console.log('[ChatPage] agent_end received, blocks_len=', streamingBlocksRef.current.length, 'committed=', messageCommittedRef.current, 'hasMessages=', !!data?.messages)
        if (streamingBlocksRef.current.length > 0) {
          const blocks = [...streamingBlocksRef.current]
          if (messageCommittedRef.current) {
            // message_end already committed — replace with agent_end's data
            setMessages(msgs => { const copy = [...msgs]; copy[copy.length - 1] = { role: 'assistant', content: blocks }; return copy })
          } else {
            setMessages(msgs => [...msgs, { role: 'assistant', content: blocks }])
            saveMessage('assistant', blocks)
            messageCommittedRef.current = true
          }
        } else if (!messageCommittedRef.current && data?.messages && Array.isArray(data.messages)) {
          // No streaming blocks built — parse final messages from agent_end payload.
          console.log('[ChatPage] agent_end: using data.messages fallback, count=', data.messages.length)
          const lastAssistantMsg = [...data.messages].reverse().find(
            (m: any) => m.role === 'assistant'
          )
          if (lastAssistantMsg?.content && Array.isArray(lastAssistantMsg.content)) {
            const blocks: ContentBlock[] = []
            for (const item of lastAssistantMsg.content) {
              if (item.type === 'thinking') {
                blocks.push({ type: 'thinking', thinking: item.thinking || '' })
              } else if (item.type === 'text') {
                blocks.push({ type: 'text', text: item.text || '' })
              } else if (item.type === 'tool_use') {
                blocks.push({
                  type: 'tool_use',
                  toolCallId: item.id || String(blocks.length),
                  toolName: item.name || 'unknown',
                  input: item.input || {},
                })
              } else if (item.type === 'tool_result') {
                // tool_result blocks carry results in content field
                blocks.push({
                  type: 'tool_use',
                  toolCallId: item.tool_use_id || item.id || String(blocks.length),
                  toolName: 'tool',
                  input: {},
                  result: typeof item.content === 'string' ? item.content : JSON.stringify(item.content),
                })
              }
            }
            if (blocks.length > 0) {
              setMessages(msgs => [...msgs, { role: 'assistant', content: blocks }])
              saveMessage('assistant', blocks)
              messageCommittedRef.current = true
            }
          }
        }
        streamingBlocksRef.current = []
        setStreamingBlocks([])
        break
      }

      case 'message_end': {
        if (data?.message?.stopReason === 'error') {
          setMessages(prev => [
            ...prev,
            { role: 'system', content: [{ type: 'text', text: 'Error: ' + (data.message.errorMessage || 'Unknown error') }] },
          ])
        } else if (streamingBlocksRef.current.length > 0) {
          const blocks = [...streamingBlocksRef.current]
          if (messageCommittedRef.current) {
            // agent_end committed first — replace with message_end's (more complete) data
            setMessages(msgs => { const copy = [...msgs]; copy[copy.length - 1] = { role: 'assistant', content: blocks }; return copy })
          } else {
            setMessages(msgs => [...msgs, { role: 'assistant', content: blocks }])
            saveMessage('assistant', blocks)
          }
          messageCommittedRef.current = true
        } else if (data?.message?.content && Array.isArray(data.message.content) && data.message.content.length > 0 && data.message.role !== 'user') {
          // No streaming happened (e.g. agent returned immediately), or agent_end
          // already consumed the streaming blocks — parse from message_end payload.
          const blocks: ContentBlock[] = []
          for (const item of data.message.content) {
            if (item.type === 'thinking') {
              blocks.push({ type: 'thinking', thinking: item.thinking || '' })
            } else if (item.type === 'text') {
              blocks.push({ type: 'text', text: item.text || '' })
            } else if (item.type === 'tool_use') {
              blocks.push({
                type: 'tool_use',
                toolCallId: item.id || String(blocks.length),
                toolName: item.name || 'unknown',
                input: item.input || {},
              })
            }
          }
          if (blocks.length > 0) {
            if (messageCommittedRef.current) {
              setMessages(msgs => { const copy = [...msgs]; copy[copy.length - 1] = { role: 'assistant', content: blocks }; return copy })
            } else {
              setMessages(msgs => [...msgs, { role: 'assistant', content: blocks }])
              saveMessage('assistant', blocks)
            }
            messageCommittedRef.current = true
          }
        }
        streamingBlocksRef.current = []
        setStreamingBlocks([])
        break
      }

      case 'error': {
        setMessages(prev => [
          ...prev,
          { role: 'system', content: [{ type: 'text', text: 'Error: ' + data.message }] },
        ])
        break
      }

      default:
        // agent_start, turn_start, turn_end, message_start and other metadata
        // events are intentionally not rendered — they are internal protocol events.
        break
    }
  }

  // ── WebSocket connection ──

  useEffect(() => {
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null
    let ws: WebSocket | null = null

    const connect = () => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const url = `${protocol}//${window.location.host}/api/instances/${id}/chat`
      ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        console.log('[ChatPage] WS connected')
        setConnected(true)
      }
      ws.onclose = () => {
        console.log('[ChatPage] WS closed')
        setConnected(false)
        reconnectTimer = setTimeout(connect, 3000)
      }
      ws.onerror = (err) => {
        console.error('[ChatPage] WS error:', err)
      }

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data)
          processEvent(msg.type, msg.data)
        } catch (err) {
          console.error('[ChatPage] WS message error:', err, event.data?.substring?.(0, 200))
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

  // ── auto-scroll ──

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingBlocks])

  // ── HTTP SSE fallback ──

  const sendViaHttp = async (trimmed: string) => {
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
      let currentEvent = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const rawLine of lines) {
          const line = rawLine.trim()
          if (!line) {
            currentEvent = ''
            continue
          }
          if (line.startsWith('event: ')) {
            currentEvent = line.substring(7).trim()
          } else if (line.startsWith('data: ')) {
            const dataStr = line.substring(6)
            try {
              const parsed = JSON.parse(dataStr)
              processEvent(currentEvent || 'text_delta', parsed)
            } catch { /* skip malformed JSON */ }
            currentEvent = ''
          }
          // Ignore SSE comments (lines starting with ":") and other fields (id, retry)
        }
      }

      // Flush any remaining buffer
      if (buffer.trim()) {
        if (buffer.startsWith('data: ')) {
          try {
            const parsed = JSON.parse(buffer.substring(6))
            processEvent(currentEvent || 'agent_end', parsed)
          } catch { /* ignore */ }
        }
      }

      // Safety net: if we still have streaming blocks, commit them
      if (streamingBlocksRef.current.length > 0 && !messageCommittedRef.current) {
        setMessages(msgs => [...msgs, { role: 'assistant', content: [...streamingBlocksRef.current] }])
        saveMessage('assistant', [...streamingBlocksRef.current])
        messageCommittedRef.current = true
      }
      streamingBlocksRef.current = []
      setStreamingBlocks([])
    } catch (e: unknown) {
      setMessages(prev => [
        ...prev,
        {
          role: 'system',
          content: [{ type: 'text', text: 'Error: ' + (e instanceof Error ? e.message : String(e)) }],
        },
      ])
    }
  }

  const isStreaming = streamingBlocks.length > 0

  // ── send handler ──

  const handleSend = () => {
    const trimmed = input.trim()
    if (!trimmed || isStreaming) {
      console.log('[ChatPage] handleSend blocked:', { trimmed: !!trimmed, isStreaming })
      return
    }

    const wsState = wsRef.current?.readyState
    console.log('[ChatPage] handleSend:', { trimmed, wsState, isOpen: wsState === WebSocket.OPEN })
    messageCommittedRef.current = false
    if (wsState !== WebSocket.OPEN) {
      setMessages(prev => [...prev, { role: 'user', content: [{ type: 'text', text: trimmed }] }])
      saveMessage('user', [{ type: 'text', text: trimmed }])
      setInput('')
      sendViaHttp(trimmed)
      return
    }

    console.log('[ChatPage] handleSend: sending via WebSocket')
    setMessages(prev => [...prev, { role: 'user', content: [{ type: 'text', text: trimmed }] }])
    saveMessage('user', [{ type: 'text', text: trimmed }])
    try {
      wsRef.current!.send(JSON.stringify({ type: 'chat', message: trimmed }))
      setInput('')
    } catch {
      sendViaHttp(trimmed)
      setInput('')
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey && !isStreaming && !e.nativeEvent.isComposing) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleStop = async () => {
    try {
      await fetch('/api/instances/' + id + '/abort', { method: 'POST' })
    } catch { /* ignore */ }
    // Commit any partial streaming content as a message
    if (streamingBlocksRef.current.length > 0) {
      const blocks = [...streamingBlocksRef.current]
      setMessages(msgs => [...msgs, { role: 'assistant', content: blocks }])
      saveMessage('assistant', blocks)
    }
    streamingBlocksRef.current = []
    setStreamingBlocks([])
  }

  // ── render helpers ──

  const renderBlock = (block: ContentBlock, blockIdx: number) => {
    switch (block.type) {
      case 'text':
        return (
          <div key={blockIdx} style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
            {block.text}
          </div>
        )

      case 'thinking':
        return (
          <details key={blockIdx} style={{ marginBottom: 8 }}>
            <summary style={{ cursor: 'pointer', color: 'var(--text-muted)', fontSize: 12, userSelect: 'none' }}>
              Thinking...
            </summary>
            <div
              style={{
                whiteSpace: 'pre-wrap',
                fontSize: 12,
                color: 'var(--text-muted)',
                marginTop: 6,
                paddingLeft: 12,
                borderLeft: '2px solid var(--border)',
              }}
            >
              {block.thinking}
            </div>
          </details>
        )

      case 'tool_use':
        return (
          <details key={blockIdx} style={{ marginBottom: 8 }}>
            <summary
              style={{
                cursor: 'pointer',
                color: 'var(--warning)',
                fontSize: 12,
                fontWeight: 600,
                userSelect: 'none',
              }}
            >
              Tool: {block.toolName}
            </summary>
            <div style={{ marginTop: 6 }}>
              <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4 }}>Input:</div>
              <pre
                style={{
                  fontSize: 11,
                  fontFamily: "'Cascadia Code', 'JetBrains Mono', 'Fira Code', monospace",
                  background: 'var(--bg)',
                  padding: 8,
                  borderRadius: 4,
                  overflow: 'auto',
                  maxHeight: 200,
                  margin: 0,
                  color: 'var(--text)',
                }}
              >
                {JSON.stringify(block.input, null, 2)}
              </pre>
              {block.result !== undefined && (
                <>
                  <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 4, marginTop: 8 }}>
                    Result:
                  </div>
                  <pre
                    style={{
                      fontSize: 11,
                      fontFamily: "'Cascadia Code', 'JetBrains Mono', 'Fira Code', monospace",
                      background: 'var(--bg)',
                      padding: 8,
                      borderRadius: 4,
                      overflow: 'auto',
                      maxHeight: 300,
                      margin: 0,
                      color: 'var(--text)',
                    }}
                  >
                    {block.result}
                  </pre>
                </>
              )}
            </div>
          </details>
        )
    }
  }

  const hasStreamingText =
    streamingBlocks.length > 0 &&
    streamingBlocks[streamingBlocks.length - 1].type !== 'tool_use'

  // ── UI ──

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 64px)' }}>
      {/* header */}
      <div
        style={{
          padding: '12px 20px',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          background: 'var(--sidebar-bg)',
        }}
      >
        <Link to="/instances" className="btn btn-ghost" style={{ fontSize: 12 }}>
          Back
        </Link>
        <h3 style={{ fontSize: 15, margin: 0 }}>Instance {id?.substring(0, 8)}...</h3>
        <span className={`tag ${connected ? 'tag-ready' : 'tag-failed'}`}>
          {connected ? 'connected' : 'disconnected'}
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

      {/* messages */}
      <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
        {messages.map((msg, i) => (
          <div
            key={i}
            style={{
              marginBottom: 16,
              display: 'flex',
              justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
            }}
          >
            <div
              style={{
                maxWidth: '80%',
                padding: '10px 16px',
                borderRadius: 'var(--radius)',
                background:
                  msg.role === 'user'
                    ? 'var(--accent)'
                    : msg.role === 'system'
                      ? 'var(--bg)'
                      : 'var(--card-bg)',
                color: msg.role === 'user' ? '#fff' : 'var(--text)',
                border: msg.role !== 'user' ? '1px solid var(--border)' : 'none',
                fontSize: 13,
                lineHeight: 1.6,
              }}
            >
              {msg.content.map((block, j) => renderBlock(block, j))}
            </div>
          </div>
        ))}

        {/* streaming bubble */}
        {streamingBlocks.length > 0 && (
          <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-start' }}>
            <div
              style={{
                maxWidth: '80%',
                padding: '10px 16px',
                borderRadius: 'var(--radius)',
                background: 'var(--card-bg)',
                border: '1px solid var(--border)',
                fontSize: 13,
                lineHeight: 1.6,
              }}
            >
              {streamingBlocks.map((block, j) => {
                // For the very last block during streaming, don't render tool_use blocks
                // without results with the cursor — the result hasn't arrived yet.
                const isLast = j === streamingBlocks.length - 1
                return (
                  <span key={j}>
                    {renderBlock(block, j)}
                    {isLast && hasStreamingText && (
                      <span
                        style={{
                          display: 'inline-block',
                          width: 8,
                          height: 14,
                          background: 'var(--accent)',
                          marginLeft: 2,
                          verticalAlign: 'middle',
                          animation: 'blink 1s step-end infinite',
                        }}
                      />
                    )}
                  </span>
                )
              })}
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* input */}
      <div
        style={{
          padding: '16px 24px',
          borderTop: '1px solid var(--border)',
          background: 'var(--sidebar-bg)',
        }}
      >
        <div style={{ display: 'flex', gap: 10 }}>
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={
              isStreaming
                ? 'Agent is responding...'
                : connected
                  ? 'Type a message... (Enter to send, Shift+Enter for new line)'
                  : 'Disconnected \u2014 start the instance to chat'
            }
            disabled={!connected || isStreaming}
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
          {isStreaming ? (
            <button onClick={handleStop} className="btn btn-danger" style={{ alignSelf: 'flex-end' }}>
              Stop
            </button>
          ) : (
            <button onClick={handleSend} disabled={!connected} className="btn btn-primary" style={{ alignSelf: 'flex-end', opacity: connected ? 1 : 0.5 }}>
              Send
            </button>
          )}
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

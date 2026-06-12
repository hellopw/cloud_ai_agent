const BASE = '/api'

async function request(path: string, options?: RequestInit) {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  return res.json()
}

export const promptsApi = {
  list: () => request('/prompts'),
  get: (id: string) => request(`/prompts/${id}`),
  create: (data: any) => request('/prompts', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/prompts/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/prompts/${id}`, { method: 'DELETE' }),
}

export const skillsApi = {
  list: () => request('/skills'),
  get: (id: string) => request(`/skills/${id}`),
  create: (data: any) => request('/skills', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/skills/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/skills/${id}`, { method: 'DELETE' }),
}

export const toolsApi = {
  list: () => request('/tools'),
  get: (id: string) => request(`/tools/${id}`),
  create: (data: any) => request('/tools', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/tools/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/tools/${id}`, { method: 'DELETE' }),
}

export const memoriesApi = {
  list: () => request('/memories'),
  get: (id: string) => request(`/memories/${id}`),
  create: (data: any) => request('/memories', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/memories/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/memories/${id}`, { method: 'DELETE' }),
}

export const templatesApi = {
  list: () => request('/templates'),
  get: (id: string) => request(`/templates/${id}`),
  create: (data: any) => request('/templates', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/templates/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/templates/${id}`, { method: 'DELETE' }),
  bind: (id: string, data: any) => request(`/templates/${id}/bind`, { method: 'PUT', body: JSON.stringify(data) }),
}

export const agentsApi = {
  list: () => request('/agents'),
  get: (id: string) => request(`/agents/${id}`),
  create: (data: any) => request('/agents', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/agents/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request(`/agents/${id}`, { method: 'DELETE' }),
}

export const instancesApi = {
  list: () => request('/instances'),
  get: (id: string) => request(`/instances/${id}`),
  delete: (id: string) => request(`/instances/${id}`, { method: 'DELETE' }),
}

export const providerConfigsApi = {
  list: () => request('/provider-configs'),
  get: (id: string) => request(`/provider-configs/${id}`),
  create: (data: any) => request('/provider-configs', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/provider-configs/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/provider-configs/${id}`, { method: 'DELETE' }),
}

export const resourcesApi = {
  list: () => request('/resources'),
  get: (id: string) => request(`/resources/${id}`),
  create: (data: any) => request('/resources', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/resources/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/resources/${id}`, { method: 'DELETE' }),
}

// Agent Teams
export const agentTeamsApi = {
  list: () => request('/agent-teams'),
  get: (id: string) => request(`/agent-teams/${id}`),
  create: (data: any) => request('/agent-teams', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/agent-teams/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/agent-teams/${id}`, { method: 'DELETE' }),
}

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

// Prompts
export const promptsApi = {
  list: () => request('/prompts'),
  get: (id: string) => request(`/prompts/${id}`),
  create: (data: any) => request('/prompts', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/prompts/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/prompts/${id}`, { method: 'DELETE' }),
}

// Skills
export const skillsApi = {
  list: () => request('/skills'),
  get: (id: string) => request(`/skills/${id}`),
  create: (data: any) => request('/skills', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/skills/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/skills/${id}`, { method: 'DELETE' }),
}

// Tools
export const toolsApi = {
  list: () => request('/tools'),
  get: (id: string) => request(`/tools/${id}`),
  create: (data: any) => request('/tools', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/tools/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/tools/${id}`, { method: 'DELETE' }),
}

// Templates
export const templatesApi = {
  list: () => request('/templates'),
  get: (id: string) => request(`/templates/${id}`),
  create: (data: any) => request('/templates', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => request(`/templates/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/templates/${id}`, { method: 'DELETE' }),
  bind: (id: string, data: any) => request(`/templates/${id}/bind`, { method: 'PUT', body: JSON.stringify(data) }),
}

// Agents
export const agentsApi = {
  list: () => request('/agents'),
  get: (id: string) => request(`/agents/${id}`),
  create: (data: any) => request('/agents', { method: 'POST', body: JSON.stringify(data) }),
  delete: (id: string) => request(`/agents/${id}`, { method: 'DELETE' }),
}

// Instances
export const instancesApi = {
  list: () => request('/instances'),
  get: (id: string) => request(`/instances/${id}`),
  delete: (id: string) => request(`/instances/${id}`, { method: 'DELETE' }),
}
